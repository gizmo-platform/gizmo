package net

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/gizmo-platform/gizmo/pkg/metrics"
	"github.com/gizmo-platform/gizmo/pkg/routeros/config"
)

// TLM is a Team Location Mapper that contains a static mapping.
type TLM struct {
	l hclog.Logger
	m mqtt.Client

	mapping    map[int]string
	mutex      sync.RWMutex
	metrics    *metrics.Metrics
	controller *config.Configurator
	mqttAddr   string

	swg  *sync.WaitGroup
	stop chan struct{}
}

// New configures the TLM with the given options.
func New(opts ...Option) *TLM {
	t := new(TLM)
	t.mapping = make(map[int]string)
	t.stop = make(chan (struct{}))
	t.mqttAddr = "mqtt://127.0.0.1:1883"

	for _, o := range opts {
		o(t)
	}

	copts := mqtt.NewClientOptions().
		AddBroker(t.mqttAddr).
		SetAutoReconnect(true).
		SetClientID("self-tlm").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	t.m = mqtt.NewClient(copts)

	return t
}

// GetFieldForTeam returns the current location for a given team number.
func (tlm *TLM) GetFieldForTeam(team int) (string, error) {
	tlm.mutex.RLock()
	mapping, ok := tlm.mapping[team]
	tlm.mutex.RUnlock()
	if !ok {
		return "none:none", errors.New("no mapping for team")
	}
	return mapping, nil
}

// InsertOnDemandMap inserts an on-demand mapping that overrides any
// current schedule.  WARNING: This is immediate.
func (tlm *TLM) InsertOnDemandMap(m map[int]string) error {
	tlm.mutex.Lock()
	defer tlm.mutex.Unlock()
	tlm.mapping = m
	tlm.metrics.ExportCurrentMatch(tlm.mapping)

	if err := tlm.controller.SyncTLM(tlm.mapping); err != nil {
		tlm.l.Error("Error syncronizing match state", "error", err)
		return err
	}

	// This is normally unsafe to do because it doesn't actually
	// check what state the fields are in before applying
	// configuration.  This is what the reconcile-net command is
	// provided for so that if you make a manual change to the
	// system its possible to reconcile that state back to what
	// the system expects is going on.  We do this because its
	// SIGNIFICANTLY faster than doing a state refresh and in the
	// general case, very few people even know how to take manual
	// control out from under the FMS in the first place.
	if err := tlm.controller.Converge(false, ""); err != nil {
		tlm.l.Error("Error converging fields", "error", err)
		return err
	}

	if err := tlm.controller.CycleRadio("2ghz"); err != nil {
		tlm.l.Error("Error cycling radios", "error", err)
		return err
	}
	return nil
}

// GetCurrentMapping is a convenience function to retrieve the current
// mapping for the current match.
func (tlm *TLM) GetCurrentMapping() (map[int]string, error) { return tlm.mapping, nil }

// GetCurrentTeams returns the teams that are expected to be on the
// field at this time.
func (tlm *TLM) GetCurrentTeams() []int {
	tlm.metrics.ClearSchedule()
	ret := make([]int, len(tlm.mapping))

	i := 0
	tlm.mutex.RLock()
	for team := range tlm.mapping {
		ret[i] = team
	}
	tlm.mutex.RUnlock()

	return ret
}

// Start starts up a pusher that publishes location information into
// the broker.
func (tlm *TLM) Start() error {
	if tok := tlm.m.Connect(); tok.Wait() && tok.Error() != nil {
		tlm.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	tlm.l.Info("Connected to broker")

	locTicker := time.NewTicker(time.Second * 5)

	go func() {
		for {
			select {
			case <-tlm.stop:
				locTicker.Stop()
				return
			case <-locTicker.C:
				tlm.mutex.RLock()
				tlm.metrics.ExportCurrentMatch(tlm.mapping)
				bytes, err := json.Marshal(tlm.mapping)
				if err != nil {
					tlm.l.Error("Error marshalling mapping", "error", err)
					tlm.mutex.RUnlock()
					return
				}
				tlm.l.Debug("Pushing locations")
				if tok := tlm.m.Publish("sys/tlm/locations", 1, false, bytes); tok.Wait() && tok.Error() != nil {
					tlm.l.Warn("Error publishing new mapping to broker", "error", tok.Error())
				}

				for team, field := range tlm.mapping {
					parts := strings.SplitN(field, ":", 2)
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
						tlm.l.Warn("Error marshalling location", "error", err, "team", team)
					}
					if tok := tlm.m.Publish(fmt.Sprintf("robot/%d/location", team), 1, false, bytes); tok.Wait() && tok.Error() != nil {
						tlm.l.Warn("Error publishing new mapping to broker", "error", tok.Error())
					}
				}
				tlm.mutex.RUnlock()
			}
		}
	}()
	tlm.swg.Done()
	return nil
}

// Stop cancels the async location pusher.
func (tlm *TLM) Stop() {
	tlm.l.Info("Stopping...")
	tlm.stop <- struct{}{}
}
