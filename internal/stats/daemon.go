package stats

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

// Report contains the information that becomes part of the statistics
// data used by the prometheus exporter.
type Report struct {
	VBat              int32
	WatchdogRemaining int32
	WatchdogOK        bool
	RSSI              uint8
	PwrBoard          bool
	PwrPico           bool
	PwrGPIO           bool
	PwrMainA          bool
	PwrMainB          bool
}

// MqttListen forms a local client on the broker which then handles
// the conversion from MQTT stats messages to updating the metric
// registries for prometheus data.
func MqttListen(connect string, metrics *Metrics, wg *sync.WaitGroup) error {
	wg.Add(1)
	opts := mqtt.NewClientOptions().
		AddBroker(connect).
		SetAutoReconnect(true).
		SetClientID("self-metrics").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		metrics.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	metrics.l.Info("Connected to broker")
	callback := func(client mqtt.Client, message mqtt.Message) {
		teamNum := strings.Split(message.Topic(), "/")[1]
		metrics.l.Trace("Called back", "team", teamNum)
		var stats Report
		json.Unmarshal(message.Payload(), &stats)
		metrics.rssi.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.RSSI))
		metrics.vbat.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.VBat))
		metrics.watchdogRemaining.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WatchdogRemaining))

		metrics.powerBoard.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrBoard))
		metrics.powerPico.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrPico))
		metrics.powerGPIO.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrGPIO))
		metrics.powerMainA.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainA))
		metrics.powerMainB.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainB))
		metrics.watchdogOK.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.WatchdogOK))
	}

	subFunc := func() error {
		if tok := client.Subscribe("robot/+/stats", 1, callback); tok.Wait() && tok.Error() != nil {
			metrics.l.Warn("Error subscribing to topic", "error", tok.Error())
			return tok.Error()
		}
		return nil
	}
	if err := backoff.Retry(subFunc, backoff.NewExponentialBackOff()); err != nil {
		metrics.l.Error("Permanent error encountered while subscribing", "error", err)
		return err
	}
	metrics.l.Info("Subscribed to topics")
	wg.Done()
	return nil
}

func fCast(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
