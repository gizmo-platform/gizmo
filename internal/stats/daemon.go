package stats

import (
	"encoding/json"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

type StatsReport struct {
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

func MqttListen(connect string, metrics *metrics) error {
	opts := mqtt.NewClientOptions().
		AddBroker(connect).
		SetAutoReconnect(true).
		SetClientID("self").
		SetConnectRetry(true)
	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		metrics.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	metrics.l.Info("Connected to broker")
	callback := func(client mqtt.Client, message mqtt.Message) {
		fmt.Println("Yo")
		teamNum := strings.Split(message.Topic(), "/")[1]
		metrics.l.Trace("Called back", "team", teamNum)
		var stats StatsReport
		json.Unmarshal(message.Payload(), &stats)
		metrics.rssi.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.RSSI))
		metrics.vbat.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.VBat))
		metrics.watchdogRemaining.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WatchdogRemaining))
		if stats.PwrBoard {
			metrics.powerBoard.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.powerBoard.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
		if stats.PwrPico {
			metrics.powerPico.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.powerPico.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
		if stats.PwrGPIO {
			metrics.powerGPIO.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.powerGPIO.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
		if stats.PwrMainA {
			metrics.powerMainA.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.powerMainA.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
		if stats.PwrMainB {
			metrics.powerMainB.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.powerMainB.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
		if stats.WatchdogOK {
			metrics.watchdogOK.With(prometheus.Labels{"team": teamNum}).Set(1)
		} else {
			metrics.watchdogOK.With(prometheus.Labels{"team": teamNum}).Set(0)
		}
	}

	// Why 1? IDK. Just doing it.
	if tok := client.Subscribe("robot/1237/stats", 1, callback); tok.Wait() && tok.Error() != nil {
		metrics.l.Warn("Error subscribing to topic", "error", tok.Error())
		return tok.Error()
	}
	metrics.l.Info("Subscribed to topics")
	return nil
}
