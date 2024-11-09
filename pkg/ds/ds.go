package ds

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/buildinfo"
	"github.com/gizmo-platform/gizmo/pkg/config"
	"github.com/gizmo-platform/gizmo/pkg/gamepad"
	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
	"github.com/gizmo-platform/gizmo/pkg/sysconf"
	"github.com/gizmo-platform/gizmo/pkg/watchdog"
)

const (
	ctrlRate = time.Millisecond * 20
	locRate  = time.Second * 3
	metaRate = time.Second * 5
)

// New returns a configured driverstation.
func New(opts ...Option) *DriverStation {
	d := new(DriverStation)
	d.l = hclog.NewNullLogger()
	d.stop = make(chan struct{})
	d.fCfg = FieldConfig{
		RadioMode:    "DS",
		RadioChannel: "1",
	}

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
	fmsAddr, fmsAvailable := ds.probeForFMS()
	ds.l.Info("FMS probe result", "available", fmsAvailable)
	if fmsAvailable {
		if err := ds.connectMQTT(fmt.Sprintf("mqtt://%s:1883", fmsAddr)); err != nil {
			ds.l.Error("Could not connect to FMS")
			return err
		}
		go func() {
			if err := ds.doMetaPublish(); err != nil {
				ds.l.Error("Error starting meta publisher", "error", err)
			}
		}()

		subFunc := func() error {
			topic := fmt.Sprintf("robot/%d/dsconf", ds.cfg.Team)
			ds.l.Debug("Subscribing to config topic", "topic", topic)
			if tok := ds.m.Subscribe(topic, 1, ds.cfgCallback); tok.Wait() && tok.Error() != nil {
				ds.l.Warn("Error subscribing to topic", "error", tok.Error())
				return tok.Error()
			}
			return nil
		}

		if err := backoff.Retry(subFunc, backoff.NewExponentialBackOff()); err != nil {
			ds.l.Error("Permanent error encountered while subscribing dsconf", "error", err)
			return err
		}

	} else {
		// FMS not available, start local services.
		go ds.doLocalBroker()
		go ds.doLocation()

		if err := ds.connectMQTT("mqtt://127.0.0.1:1883"); err != nil {
			ds.l.Error("Error linking MQTT", "error", err)
			ds.stop <- struct{}{}
		}
	}

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

// probeForFMS tests to see if the FMS is available by checking for
// its magic DNS names.
func (ds *DriverStation) probeForFMS() (string, bool) {
	const timeout = 50 * time.Millisecond
	var r net.Resolver

	ctx, cancel1 := context.WithTimeout(context.Background(), timeout)
	defer cancel1()
	_, err := r.LookupHost(ctx, "nxdomain.gizmo")
	if err == nil {
		ds.l.Warn("nxdomain.gizmo resolves; DNS on this network is unreliable!")
		// If no error occured here then it means that the
		// domain that is explicitly supposed to nxdomain
		// didn't, and we can't trust DNS on this network.
		return "", false
	}
	ds.l.Debug("nxdomain.gizmo nxdomains")

	ctx, cancel2 := context.WithTimeout(context.Background(), timeout)
	defer cancel2()
	res, err := r.LookupHost(ctx, "fms.gizmo")
	if err != nil {
		ds.l.Warn("FMS Unavailable", "error", err)
		return "", false
	}

	return res[0], true
}

func (ds *DriverStation) connectMQTT(address string) error {
	copts := mqtt.NewClientOptions().
		AddBroker(address).
		SetAutoReconnect(true).
		SetClientID(fmt.Sprintf("gizmo-ds%d", ds.cfg.Team)).
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	ds.m = mqtt.NewClient(copts)

	if tok := ds.m.Connect(); tok.Wait() && tok.Error() != nil {
		ds.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	ds.l.Info("Connected to broker", "broker", address)
	return nil
}

func (ds *DriverStation) doLocalBroker() error {
	m, err := mqttserver.NewServer(
		mqttserver.WithLogger(ds.l),
	)
	if err != nil {
		ds.l.Error("Error during broker initialization", "error", err)
		return err
	}

	go func() {
		if err := m.Serve(":1883"); err != nil {
			ds.l.Error("Error setting up local broker", "error", err)
			ds.stop <- struct{}{}
			return
		}
	}()

	<-ds.stop
	m.Shutdown()
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
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing location data")
			return nil
		case <-ticker.C:
			dog.Feed()
			vals := struct {
				Field    int
				Quadrant string
			}{
				Field:    1,
				Quadrant: "PRACTICE",
			}

			bytes, err := json.Marshal(vals)
			if err != nil {
				ds.l.Warn("Error marshalling controller state", "error", err)
				return err
			}

			topic := path.Join("robot", strconv.Itoa(ds.cfg.Team), "location")
			if tok := ds.m.Publish(topic, 0, false, bytes); tok.Wait() && tok.Error() != nil {
				ds.l.Warn("Error publishing message for team", "error", tok.Error())
			}
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

			bytes, err := json.Marshal(vals)
			if err != nil {
				ds.l.Warn("Error marshalling controller state", "error", err)
				return err
			}

			topic := path.Join("robot", strconv.Itoa(ds.cfg.Team), "gamepad")
			if tok := ds.m.Publish(topic, 0, false, bytes); tok.Wait() && tok.Error() != nil {
				ds.l.Warn("Error publishing message for team", "error", tok.Error())
			}
		}
	}

	return nil
}

func (ds *DriverStation) doMetaPublish() error {
	ticker := time.NewTicker(metaRate)

	vals := &config.DSMeta{
		Version:  buildinfo.Version,
		Bootmode: os.Getenv("GIZMO_BOOTMODE"),
	}

	if vals.Bootmode == "" {
		vals.Bootmode = "UNKNOWN"
	}

	bytes, err := json.Marshal(vals)
	if err != nil {
		ds.l.Warn("Error marshalling controller state", "error", err)
		return err
	}

	ds.l.Info("Starting metadata pusher")
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing metadata")
			return nil
		case <-ticker.C:

			topic := path.Join("robot", strconv.Itoa(ds.cfg.Team), "ds-meta")
			if tok := ds.m.Publish(topic, 0, false, bytes); tok.Wait() && tok.Error() != nil {
				ds.l.Warn("Error publishing message for team", "error", tok.Error())
			}
		}
	}

	return nil
}

func (ds *DriverStation) cfgCallback(c mqtt.Client, msg mqtt.Message) {
	fCfg := FieldConfig{}

	ds.l.Debug("Config Callback Called")

	if err := json.Unmarshal(msg.Payload(), &fCfg); err != nil {
		ds.l.Warn("Bad config payload", "error", err)
		return
	}

	if ds.fCfg.RadioChannel != fCfg.RadioChannel || ds.fCfg.RadioMode != fCfg.RadioMode {
		ds.fCfg = fCfg
		if err := ds.reconfigureRadio(); err != nil {
			ds.l.Error("Error reconfiguring radio", "error", err)
		}
	}
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
