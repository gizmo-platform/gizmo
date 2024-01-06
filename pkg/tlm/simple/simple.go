package simple

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"

	"github.com/bestrobotics/gizmo/pkg/metrics"
)

// TLM is a Team Location Mapper that contains a static mapping.
type TLM struct {
	l hclog.Logger
	m mqtt.Client

	mapping  map[int]string
	mutex    sync.RWMutex
	mqttAddr string
	metrics  *metrics.Metrics

	swg *sync.WaitGroup

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
		SetClientID("self").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	client := mqtt.NewClient(copts)
	t.m = client

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

// SetScheduleStep normally would advance the schedule when running a
// scheduled match.
func (tlm *TLM) SetScheduleStep(_ int) error { return nil }

// InsertOnDemandMap inserts an on-demand mapping that overrides any
// current schedule.  WARNING: This is immediate.
func (tlm *TLM) InsertOnDemandMap(m map[int]string) {

	tlm.mutex.Lock()
	tlm.mapping = m
	tlm.mutex.Unlock()
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
				tlm.mutex.RUnlock()
				if tok := tlm.m.Publish("sys/tlm/locations", 1, false, bytes); tok.Wait() && tok.Error() != nil {
					tlm.l.Warn("Error publishing new mapping to broker", "error", tok.Error())
				}
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
