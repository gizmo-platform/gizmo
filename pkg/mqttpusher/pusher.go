package mqttpusher

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/bestrobotics/gizmo/pkg/gamepad"
)

// JSController defines the interface that the control server expects
// to be able to serve
type JSController interface {
	GetState(string) (*gamepad.Values, error)
}

// Pusher connects to the broker and pushes joystick data out to the
// robots per the internal mapping.
type Pusher struct {
	l hclog.Logger
	m mqtt.Client

	addr        string
	teams       map[int]string
	tMutex      sync.RWMutex
	pushWorkers map[int]chan struct{}
	locWorkers  map[int]chan struct{}

	locRate  time.Duration
	ctrlRate time.Duration

	// Map of quad/fid to gamepad ID
	quadMap map[string]int

	swg *sync.WaitGroup
}

// New configures and returns a connected pusher.
func New(opts ...Option) (*Pusher, error) {
	p := new(Pusher)
	p.swg = new(sync.WaitGroup)
	p.teams = make(map[int]string)
	p.pushWorkers = make(map[int]chan struct{})
	p.locWorkers = make(map[int]chan struct{})
	p.quadMap = make(map[string]int)
	p.ctrlRate = time.Millisecond * 20
	p.locRate = time.Second * 3

	for _, o := range opts {
		if err := o(p); err != nil {
			return nil, err
		}
	}

	copts := mqtt.NewClientOptions().
		AddBroker(p.addr).
		SetAutoReconnect(true).
		SetClientID("self-pusher").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	client := mqtt.NewClient(copts)
	p.m = client
	return p, nil
}

// Connect allows for setting up the connection later, after the
// pusher is initialized.
func (p *Pusher) Connect() error {
	if tok := p.m.Connect(); tok.Wait() && tok.Error() != nil {
		p.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	p.l.Info("Connected to broker")

	subFunc := func() error {
		if tok := p.m.Subscribe("sys/tlm/locations", 1, p.updateLoc); tok.Wait() && tok.Error() != nil {
			p.l.Warn("Error subscribing to topic", "error", tok.Error())
			return tok.Error()
		}
		p.l.Info("Subscribed to topics")
		return nil
	}
	if err := backoff.Retry(subFunc, backoff.NewExponentialBackOff()); err != nil {
		p.l.Error("Permanent error encountered while subscribing", "error", err)
		return err
	}
	p.swg.Done()

	return nil
}

func (p *Pusher) publishGamepadForTeam(team int, fid string, stop chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	jsc := gamepad.NewJSController(gamepad.WithLogger(p.l))
	jsID, ok := p.quadMap[fid]
	if !ok {
		p.l.Error("Trying to bind a quad that doesn't exist!", "fid", fid)
		return
	}
	if err := jsc.BindController(jsID); err != nil {
		p.l.Error("Error binding gamepad!", "error", err, "team", team, "fid", fid)
		return
	}
	defer jsc.Close()

	ticker := time.NewTicker(p.ctrlRate)

	for {
		select {
		case <-stop:
			ticker.Stop()
			p.l.Info("Stopped publishing control data", "team", team, "fid", fid)
			return
		case <-ticker.C:
			vals, err := jsc.GetState()
			if err != nil {
				p.l.Warn("Error retrieving controller state", "team", team, "fid", fid, "error", err)
				retryFunc := func() error {
					if err := jsc.Rebind(); err != nil {
						p.l.Warn("Rebind failed", "team", team, "fid", fid, "error", err)
						return err
					}
					return nil
				}
				if err := backoff.Retry(retryFunc, backoff.NewExponentialBackOff()); err != nil {
					p.l.Error("Permanent error encountered while rebinding", "error", err)
				}
				return
			}

			bytes, err := json.Marshal(vals)
			if err != nil {
				p.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
				return
			}

			topic := path.Join("robot", fmt.Sprintf("%04d", team), "gamepad")
			if tok := p.m.Publish(topic, 0, false, bytes); tok.Wait() && tok.Error() != nil {
				p.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
			}
		}
	}
}

func (p *Pusher) publishLocationForTeam(team int, fid string, stop chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	parts := strings.SplitN(fid, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
	vals := struct {
		Field    int
		Quadrant string
	}{
		Field:    fnum,
		Quadrant: strings.ToUpper(parts[1]),
	}

	ticker := time.NewTicker(p.locRate)

	for {
		select {
		case <-stop:
			ticker.Stop()
			p.l.Info("Stopped announcing location", "team", team, "field", fnum, "quad", vals.Quadrant)
			return
		case <-ticker.C:
			bytes, err := json.Marshal(vals)
			if err != nil {
				p.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
				return
			}

			topic := path.Join("robot", fmt.Sprintf("%04d", team), "location")
			if tok := p.m.Publish(topic, 0, false, bytes); tok.Wait() && tok.Error() != nil {
				p.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
			}
		}
	}
}

func (p *Pusher) updateLoc(c mqtt.Client, message mqtt.Message) {
	p.tMutex.Lock()
	defer p.tMutex.Unlock()
	update := make(map[int]string)
	if err := json.Unmarshal(message.Payload(), &update); err != nil {
		p.l.Error("Error unmarshalling location data", "error", err)
	}

	for team := range p.teams {
		// Check if the team is not in the update, and if not
		// shut down the streams for them.
		if _, active := update[team]; !active {
			close(p.pushWorkers[team])
			close(p.locWorkers[team])
			delete(p.teams, team)
		}
	}

	for team, quad := range update {
		// Check if we're already handling this team and if
		// not handle them.
		if _, active := p.teams[team]; active {
			continue
		}
		pw := make(chan struct{})
		p.pushWorkers[team] = pw
		p.teams[team] = quad
		go p.publishGamepadForTeam(team, quad, pw, p.swg)

		lw := make(chan struct{})
		p.locWorkers[team] = lw
		go p.publishLocationForTeam(team, quad, lw, p.swg)
	}

	p.l.Debug("Updated pusher location information", "location", p.teams)
}

// Stop closes down the workers that publish information into the mqtt
// streams.
func (p *Pusher) Stop() {
	p.l.Info("Stopping...")
	for team := range p.teams {
		close(p.pushWorkers[team])
		close(p.locWorkers[team])
	}
	p.swg.Wait()
}
