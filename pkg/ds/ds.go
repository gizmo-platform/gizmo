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
// of FMS services locally.
func (ds *DriverStation) Run() error {
	for !ds.quit {
		ds.stop = make(chan struct{})
		go ds.findFMS()
		if !ds.fmsAvailable {
			// We're running standalone so spin up the local
			// broker and location service.
			go ds.doLocalBroker()
			go ds.doLocation()

			// Bring up the local radio
			ds.svc.Start("hostapd")
		} else {
			// The FMS exists at a special DNS alias, if that
			// resolved, we can connect to it.
			ds.connectMQTT("mqtt://fms.gizmo:1883")

			// Shut down the local radio
			ds.svc.Stop("hostapd")
		}
		go ds.doGamepad()
		<-ds.stop
	}
	return nil
}

// Stop shuts down all the various components and requests the driver
// station to stop.
func (ds *DriverStation) Stop() {
	ds.l.Info("Receieved stop request")
	ds.quit = true
	close(ds.stop)
}

func (ds *DriverStation) findFMS() {
	var r net.Resolver
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ds.stop:
			t.Stop()
		case <-t.C:
			ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*50)
			_, err := r.LookupHost(ctx, "fms.gizmo")
			ds.l.Trace("FMS Detection result", "error", err)
			available := err == nil

			if ds.fmsAvailable != available {
				ds.l.Debug("FMS Availability changed", "available", available)
				close(ds.stop)
			}
			ds.fmsAvailable = available
		}
	}
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
	ds.l.Info("Connected to broker")
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
