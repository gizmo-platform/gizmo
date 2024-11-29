package ds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/gamepad"
	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/sysconf"
	"github.com/gizmo-platform/gizmo/pkg/watchdog"
)

const (
	ctrlRate = time.Millisecond * 25
	locRate  = time.Second * 3
	metaRate = time.Second * 5
	cfgRate  = time.Second * 5
)

// New returns a configured driverstation.
func New(opts ...Option) *DriverStation {
	d := new(DriverStation)
	d.l = hclog.NewNullLogger()
	d.stop = make(chan struct{})
	d.localFieldConfig()

	for _, o := range opts {
		o(d)
	}

	d.sc = sysconf.New(sysconf.WithFS(efs), sysconf.WithLogger(d.l))
	return d
}

// Run starts up the processes that actually push control data, and if
// no external FMS is detected then it will also run the bare minimum
// of FMS services locally.  The FMS detection happens once per
// startup, because every time the network link is cycled the DS
// process gets restarted.
func (ds *DriverStation) Run() error {
	ds.reconfigureRadio()

	var err error
	ds.c, err = net.Dial("udp", fmt.Sprintf("10.%d.%d.3:1729", int(ds.cfg.Team/100), int(ds.cfg.Team%100)))
	if err != nil {
		ds.l.Error("Error opening UDP socket", "error", err)
		return err
	}

	go ds.doLocation()
	go ds.udpServlet()
	go ds.doFMSLifecycle()

	ds.doGamepad()
	return nil
}

// Stop shuts down all the various components and requests the driver
// station to stop.
func (ds *DriverStation) Stop() {
	ds.l.Info("Receieved stop request")
	ds.quit = true
	close(ds.stop)
}

// DieNow forces an immediate exit without cleaning up references.
// This may have side effects!
func (ds *DriverStation) DieNow() {
	ds.l.Error("Told to Die!")
	os.Exit(2)
}

func (ds *DriverStation) localFieldConfig() {
	ds.fCfg = FieldConfig{
		RadioMode:    "DS",
		RadioChannel: "1",
		Field:        1,
		Location:     "PRACTICE",
	}
}

func (ds *DriverStation) udpServlet() error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(10, byte(ds.cfg.Team/100), byte(ds.cfg.Team%100), 2),
		Port: 1729,
	})
	if err != nil {
		ds.l.Error("Error binding UDP socket", "error", err)
		return err
	}

	metrics := metrics.New(metrics.WithLogger(ds.l))

	go metrics.BuiltinWebserver(":8080")
	go metrics.StartFlusher()
	go func() {
		<-ds.stop
		metrics.Shutdown()
		conn.Close()
	}()

	buf := make([]byte, 1024)
	ds.l.Info("Starting UDP Servlet")
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			ds.l.Warn("Error reading packet from UDP", "error", err)
			continue
		}

		switch rune(buf[0]) {
		case 'S':
			// Status Report
			metrics.ParseReport(fmt.Sprintf("%d", ds.cfg.Team), buf[1:n])
		case 'M':
			// BuildInfo Report
			ds.gizmoMetaCallback(buf[1:n])
		}
	}

	return nil
}

func (ds *DriverStation) doLocation() error {
	dog := watchdog.New(
		watchdog.WithName("location"),
		watchdog.WithFoodDuration(time.Second*10),
		watchdog.WithHandFunction(ds.DieNow),
		watchdog.WithLogger(ds.l),
	)

	ticker := time.NewTicker(locRate)
	buf := new(bytes.Buffer)
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing location data")
			return nil
		case <-ticker.C:
			dog.Feed()
			ds.l.Trace("Location Tick")
			vals := struct {
				Field    int
				Quadrant string
			}{
				Field:    ds.fCfg.Field,
				Quadrant: ds.fCfg.Location,
			}

			buf.WriteRune('L')
			if err := json.NewEncoder(buf).Encode(vals); err != nil {
				ds.l.Debug("Error pushing location", "error", err)
			}
			buf.WriteTo(ds.c)
			buf.Reset()
		}
	}
	return nil
}

func (ds *DriverStation) doGamepad() error {
	dog := watchdog.New(
		watchdog.WithName("gamepad"),
		watchdog.WithFoodDuration(time.Second),
		watchdog.WithHandFunction(ds.DieNow),
		watchdog.WithLogger(ds.l),
	)

	jsc := gamepad.NewJSController(gamepad.WithLogger(ds.l))
	retryFunc := func() error {
		if err := jsc.Rebind(); err != nil {
			ds.l.Warn("Rebind failed", "error", err)
			return err
		}
		return nil
	}

	if err := jsc.BindController(0); err != nil {
		ds.l.Error("Error binding gamepad!", "error", err)
		if err := backoff.Retry(retryFunc, backoff.NewConstantBackOff(time.Second*3)); err != nil {
			ds.l.Error("Permanent error encountered while rebinding", "error", err)
			return err
		}
	}
	defer jsc.Close()

	ticker := time.NewTicker(ctrlRate)
	ds.l.Info("Starting gamepad pusher")
	buf := new(bytes.Buffer)
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing control data")
			return nil
		case <-ticker.C:
			ds.l.Trace("Control loop tick")
			dog.Feed()
			vals, err := jsc.GetState()
			if err != nil {
				ds.l.Warn("Error retrieving controller state", "error", err)
				if err := backoff.Retry(retryFunc, backoff.NewConstantBackOff(time.Second*3)); err != nil {
					ds.l.Error("Permanent error encountered while rebinding", "error", err)
					return err
				}
			}

			buf.WriteRune('C')
			if err := json.NewEncoder(buf).Encode(vals); err != nil {
				ds.l.Debug("Error publishing message for team", "error", err)
			}
			buf.WriteTo(ds.c)
			buf.Reset()
		}
	}

	return nil
}

func (ds *DriverStation) doFMSLifecycle() error {
	cl := &http.Client{Timeout: time.Second}

	req := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:8080", ds.cfg.FieldIP),
			Path:   fmt.Sprintf("/gizmo/ds/%d/config", ds.cfg.Team),
		},
	}

	ticker := time.NewTicker(cfgRate)
	ds.l.Info("Starting FMS lifecycle watcher")
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped config lifecycle watcher")
			return nil

		case <-ticker.C:
			resp, err := cl.Do(req)
			if err != nil {
				ds.l.Trace("Error calling FMS config endpoint", "error", err)
				continue
			}
			if resp.StatusCode != 200 {
				ds.l.Trace("Wrong code from FMS config endpoint", "code", resp.StatusCode)
				continue
			}
			if err := ds.cfgCallback(resp.Body); err != nil {
				ds.l.Error("Error parsing config from FMS", "error", err)
				continue
			}
			ticker.Stop()
			go ds.doMetaReport()
		}
	}
}

func (ds *DriverStation) doMetaReport() error {
	cl := &http.Client{Timeout: time.Second}
	reportURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:8080", ds.cfg.FieldIP),
		Path:   fmt.Sprintf("/gizmo/ds/%d/meta", ds.cfg.Team),
	}

	ticker := time.NewTicker(metaRate)
	ds.l.Info("Starting Metadata Reporter")
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped metareport reporter")
			return nil
		case <-ticker.C:
			vals := &config.DSMeta{
				Version:  buildinfo.Version,
				Bootmode: os.Getenv("GIZMO_BOOTMODE"),
			}

			if vals.Bootmode == "" {
				vals.Bootmode = "UNKNOWN"
			}

			data, err := json.Marshal(vals)
			if err != nil {
				ds.l.Debug("Error marshalling controller state", "error", err)
				continue
			}

			req, _ := http.NewRequest(http.MethodPost, reportURL.String(), bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			_, err = cl.Do(req)
			if err != nil {
				ds.l.Debug("Could not report meta information", "error", err)
				continue
			}
		}
	}
}

func (ds *DriverStation) gizmoMetaCallback(buf []byte) {
	c := &http.Client{Timeout: time.Second}
	reportURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:8080", ds.cfg.FieldIP),
		Path:   fmt.Sprintf("/gizmo/robot/%d/meta", ds.cfg.Team),
	}
	req, _ := http.NewRequest(http.MethodPost, reportURL.String(), bytes.NewBuffer(buf))
	if _, err := c.Do(req); err != nil {
		ds.l.Warn("Could not report gizmo metadata", "error", err)
	}
}

func (ds *DriverStation) cfgCallback(cfgSrc io.ReadCloser) error {
	defer cfgSrc.Close()
	fCfg := FieldConfig{}

	ds.l.Debug("Config Callback Called")

	if err := json.NewDecoder(cfgSrc).Decode(&fCfg); err != nil {
		ds.l.Warn("Bad config payload", "error", err)
		return err
	}
	ds.l.Info("FMS Config", "radio-mode", fCfg.RadioMode, "radio-channel", fCfg.RadioChannel, "field", fCfg.Field, "quadrant", fCfg.Location)

	if ds.fCfg.RadioChannel != fCfg.RadioChannel || ds.fCfg.RadioMode != fCfg.RadioMode {
		ds.fCfg = fCfg
		if err := ds.reconfigureRadio(); err != nil {
			ds.l.Error("Error reconfiguring radio", "error", err)
			return err
		}
	}
	return nil
}

func (ds *DriverStation) reconfigureRadio() error {
	channel := ds.fCfg.RadioChannel

	if channel == "Auto" {
		rand.Seed(time.Now().UnixNano())
		chans := []string{"1", "6", "11"}
		chanIdx := rand.Intn(len(chans))
		channel = chans[chanIdx]
	}

	ds.l.Info("Reconfiguring DS Radio", "mode", ds.fCfg.RadioMode, "channel", channel)
	ctx := map[string]string{
		"NetSSID": ds.cfg.NetSSID,
		"NetPSK":  ds.cfg.NetPSK,
		"Channel": channel,
	}
	if err := ds.sc.Template(hostAPdConf, "tpl/hostapd.conf.tpl", 0644, ctx); err != nil {
		return err
	}

	switch ds.fCfg.RadioMode {
	case "DS":
		ds.sc.Restart("hostapd")
	case "FIELD":
		ds.sc.Stop("hostapd")
	case "NONE":
		ds.sc.Stop("hostapd")
	}

	return nil
}
