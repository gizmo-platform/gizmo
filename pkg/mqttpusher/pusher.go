package mqttpusher

import (
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
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

	jsc JSController

	addr   string
	teams  map[int]string
	tMutex sync.RWMutex

	stopControlFeed  chan struct{}
	stopLocationFeed chan struct{}
}

// New configures and returns a connected pusher.
func New(opts ...Option) (*Pusher, error) {
	p := new(Pusher)
	p.stopControlFeed = make(chan (struct{}))
	p.stopLocationFeed = make(chan (struct{}))
	p.teams = make(map[int]string)

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
		if tok := p.m.Subscribe("sys/tlm/locations", 1, p.updateLoc) ; tok.Wait() && tok.Error() != nil {
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

	return nil
}

func (p *Pusher) publishGamepadForTeam(team int) {
	p.tMutex.RLock()
	fid, ok := p.teams[team]
	p.tMutex.RUnlock()
	if !ok {
		p.l.Warn("Trying to send gamepad state for an unmapped team!", "team", team)
		return
	}

	vals, err := p.jsc.GetState(fid)
	if err != nil {
		p.l.Warn("Error retrieving controller state", "team", team, "fid", fid, "error", err)
		return
	}

	bytes, err := json.Marshal(vals)
	if err != nil {
		p.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
		return
	}

	topic := path.Join("robot", strconv.Itoa(team), "gamepad")
	if tok := p.m.Publish(topic, 1, false, bytes); tok.Wait() && tok.Error() != nil {
		p.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
	}
}

func (p *Pusher) publishLocationForTeam(team int) {
	p.tMutex.RLock()
	fid, ok := p.teams[team]
	p.tMutex.RUnlock()
	if !ok {
		p.l.Warn("Trying to send location for an unmapped team!", "team", team)
		return
	}

	parts := strings.SplitN(fid, ":", 2)
	fnum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
	vals := struct {
		Field    int
		Quadrant string
	}{
		Field:    fnum,
		Quadrant: strings.ToUpper(parts[1]),
	}

	bytes, err := json.Marshal(vals)
	if err != nil {
		p.l.Warn("Error marshalling controller state", "team", team, "fid", fid, "error", err)
		return
	}

	topic := path.Join("robot", strconv.Itoa(team), "location")
	if tok := p.m.Publish(topic, 1, false, bytes); tok.Wait() && tok.Error() != nil {
		p.l.Warn("Error publishing message for team", "error", err, "team", team, "fid", fid)
	}
}

func (p *Pusher) updateLoc(c mqtt.Client, message mqtt.Message) {
	p.tMutex.Lock()
	defer p.tMutex.Unlock()
	if err := json.Unmarshal(message.Payload(), &p.teams); err != nil {
		p.l.Error("Error unmarshalling location data", "error", err)
	}
	p.l.Debug("Updated pusher location information", "location", p.teams)
}

// StartLocationPusher starts up a pusher that publishes location
// information into the broker.
func (p *Pusher) StartLocationPusher() {
	locTicker := time.NewTicker(time.Second * 5)

	go func() {
		for {
			select {
			case <-p.stopLocationFeed:
				locTicker.Stop()
				return
			case <-locTicker.C:
				p.tMutex.RLock()
				teams := []int{}
				for t := range p.teams {
					teams = append(teams, t)
				}
				p.tMutex.RUnlock()
				for _, t := range teams {
					p.l.Debug("Updating location for team", "team", t)
					p.publishLocationForTeam(t)
				}

			}
		}
	}()
}

// StartControlPusher starts up a pusher that publishes control
// information into the broker.
func (p *Pusher) StartControlPusher() {
	ctrlTicker := time.NewTicker(time.Millisecond * 40)

	go func() {
		for {
			select {
			case <-p.stopControlFeed:
				ctrlTicker.Stop()
				return
			case <-ctrlTicker.C:
				p.tMutex.RLock()
				teams := []int{}
				for t := range p.teams {
					teams = append(teams, t)
				}
				p.tMutex.RUnlock()
				for _, t := range teams {
					p.publishGamepadForTeam(t)
				}
			}
		}
	}()
}

// Stop closes down the workers that publish information into the mqtt
// streams.
func (p *Pusher) Stop() {
	p.l.Info("Stopping...")
	p.stopControlFeed <- struct{}{}
	p.stopLocationFeed <- struct{}{}
}
