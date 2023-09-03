package mqttpusher

import (
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/bestfield/pkg/gamepad"
)

// JSController defines the interface that the control server expects
// to be able to serve
type JSController interface {
	GetState(string) (*gamepad.Values, error)
}

// TeamLocationMapper looks at all teams trying to fetch a value and
// tries to get them controller based on their current match and their
// number.
type TeamLocationMapper interface {
	GetCurrentTeams() []int
	GetFieldForTeam(int) (string, error)
}

// Pusher connects to the broker and pushes joystick data out to the
// robots per the internal mapping.
type Pusher struct {
	l hclog.Logger
	m mqtt.Client

	addr string

	tlm TeamLocationMapper
	jsc JSController

	stopControlFeed  chan struct{}
	stopLocationFeed chan struct{}
}

// New configures and returns a connected pusher.
func New(opts ...Option) (*Pusher, error) {
	p := new(Pusher)
	p.stopControlFeed = make(chan (struct{}))
	p.stopLocationFeed = make(chan (struct{}))

	copts := mqtt.NewClientOptions().
		AddBroker(p.addr).
		SetAutoReconnect(true).
		SetClientID("self").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	client := mqtt.NewClient(copts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		p.l.Error("Error connecting to broker", "error", tok.Error())
		return nil, tok.Error()
	}
	p.m = client
	p.l.Info("Connected to broker")
	return p, nil
}

func (p *Pusher) publishGamepadForTeam(team int) {
	fid, err := p.tlm.GetFieldForTeam(team)
	if err != nil {
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
	fid, err := p.tlm.GetFieldForTeam(team)
	if err != nil {
		p.l.Warn("Trying to send gamepad state for an unmapped team!", "team", team)
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
				for _, t := range p.tlm.GetCurrentTeams() {
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
				for _, t := range p.tlm.GetCurrentTeams() {
					p.publishGamepadForTeam(t)
				}
			}
		}
	}()
}

// Stop closes down the workers that publish information into the mqtt
// streams.
func (p *Pusher) Stop() {
	p.stopControlFeed <- struct{}{}
	p.stopLocationFeed <- struct{}{}
}
