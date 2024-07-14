package ds

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/gamepad"
	"github.com/gizmo-platform/gizmo/pkg/mqttserver"
)

const (
	ctrlRate = time.Millisecond * 20
	locRate  = time.Second * 3
)

// New returns a configured driverstation.
func New(opts ...Option) *DriverStation {
	d := new(DriverStation)
	d.l = hclog.NewNullLogger()
	d.svc = new(Runit)
	d.stop = make(chan struct{})

	for _, o := range opts {
		o(d)
	}
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
	} else {
		// FMS not available, start local services.
		go ds.doLocalBroker()
		go ds.doLocation()
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

	ds.connectMQTT("mqtt://localhost:1883")
	<-ds.stop
	m.Shutdown()

	return nil
}

func (ds *DriverStation) doLocation() error {
	ticker := time.NewTicker(locRate)
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing location data")
			return nil
		case <-ticker.C:
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
	for {
		select {
		case <-ds.stop:
			ticker.Stop()
			ds.l.Info("Stopped publishing control data")
			return nil
		case <-ticker.C:
			vals, err := jsc.GetState()
			if err != nil {
				ds.l.Warn("Error retrieving controller state", "error", err)
				if err := backoff.Retry(retryFunc, backoff.NewConstantBackOff(time.Second*3)); err != nil {
					ds.l.Error("Permanent error encountered while rebinding", "error", err)
				}
				return err
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
