package metrics

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
)

// New returns an initialized instance of the metrics system.
func New(opts ...Option) *Metrics {
	x := &Metrics{
		l:               hclog.NewNullLogger(),
		r:               prometheus.NewRegistry(),
		broker:          "mqtt://127.0.0.1:1883",
		stopStatFlusher: make(chan (struct{})),

		robotRSSI: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "rssi",
			Help:      "WiFi signal strength as measured by the system processor.",
		}, []string{"team"}),

		robotWifiReconnects: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "wifi_reconnects",
			Help:      "Number of reconnects since last boot",
		}, []string{"team"}),

		robotVBat: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "battery_voltage",
			Help:      "Robot Battery volage.",
		}, []string{"team"}),

		robotPowerBoard: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "power_board",
			Help:      "General logic power available.",
		}, []string{"team"}),

		robotPowerPico: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "power_pico",
			Help:      "Pico power supply available.",
		}, []string{"team"}),

		robotPowerGPIO: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "power_gpio",
			Help:      "GPIO power supply available.",
		}, []string{"team"}),

		robotPowerBusA: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "power_bus_a",
			Help:      "Motor Bus A power available.",
		}, []string{"team"}),

		robotPowerBusB: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "power_bus_b",
			Help:      "Motor Bus B power available.",
		}, []string{"team"}),

		robotWatchdogOK: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "watchdog_ok",
			Help:      "Watchdog has been fed and is alive.",
		}, []string{"team"}),

		robotWatchdogLifetime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "watchdog_remaining_seconds",
			Help:      "Watchdog lifetime remaining since last feed.",
		}, []string{"team"}),

		robotControlFrameAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "robot",
			Name:      "control_frame_age_seconds",
			Help:      "Time since last control frame was received",
		}, []string{"team"}),

		robotOnField: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "best",
			Subsystem: "field",
			Name:      "robot",
		}, []string{"field", "quad"}),
	}

	x.r.MustRegister(x.robotRSSI)
	x.r.MustRegister(x.robotWifiReconnects)
	x.r.MustRegister(x.robotVBat)
	x.r.MustRegister(x.robotPowerBoard)
	x.r.MustRegister(x.robotPowerPico)
	x.r.MustRegister(x.robotPowerGPIO)
	x.r.MustRegister(x.robotPowerBusA)
	x.r.MustRegister(x.robotPowerBusB)
	x.r.MustRegister(x.robotWatchdogOK)
	x.r.MustRegister(x.robotWatchdogLifetime)
	x.r.MustRegister(x.robotControlFrameAge)
	x.r.MustRegister(x.robotOnField)

	for _, o := range opts {
		o(x)
	}

	return x
}

// Registry provides access to the registry that this instance
// manages.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.r
}

// ResetRobotMetrics clears all metrics associated with robots and
// resets the built-in exporter to a clean state.
func (m *Metrics) ResetRobotMetrics() {
	m.robotRSSI.Reset()
	m.robotWifiReconnects.Reset()
	m.robotVBat.Reset()
	m.robotPowerBoard.Reset()
	m.robotPowerPico.Reset()
	m.robotPowerGPIO.Reset()
	m.robotPowerBusA.Reset()
	m.robotPowerBusB.Reset()
	m.robotWatchdogOK.Reset()
	m.robotWatchdogLifetime.Reset()
	m.robotControlFrameAge.Reset()
}

// ClearSchedule resets the status of what teams are on what fields.
func (m *Metrics) ClearSchedule() {
	m.robotOnField.Reset()
}

// ExportCurrentMatch unpacks a mapping into something exportable by
// labelling the field and quadrant on an exported metric that has the
// team number as its value.
func (m *Metrics) ExportCurrentMatch(match map[int]string) {
	for team, quad := range match {
		m.l.Debug("Exporting match", "team", team, "quad", quad)
		parts := strings.Split(quad, ":")
		field := strings.Replace(parts[0], "field", "", 1)
		quad := parts[1]

		m.robotOnField.With(prometheus.Labels{
			"field": field,
			"quad":  quad,
		}).Set(float64(team))
	}
}

func (m *Metrics) mqttCallback(c mqtt.Client, msg mqtt.Message) {
	teamNum := strings.Split(msg.Topic(), "/")[1]
	m.l.Trace("Called back", "team", teamNum)
	var stats report
	if err := json.Unmarshal(msg.Payload(), &stats); err != nil {
		m.l.Warn("Bad stats report", "team", teamNum, "error", err)
	}

	// Determined by experimental sampling with regression.
	// R^2=0.9995
	voltage := 0.008848*float64(stats.VBat) - 0.30915

	m.robotRSSI.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.RSSI))
	m.robotWifiReconnects.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WifiReconnects))
	m.robotVBat.With(prometheus.Labels{"team": teamNum}).Set(voltage)
	m.robotWatchdogLifetime.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WatchdogRemaining) / 1000)
	m.robotControlFrameAge.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.ControlFrameAge) / 1000)

	m.robotPowerBoard.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrBoard))
	m.robotPowerPico.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrPico))
	m.robotPowerGPIO.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrGPIO))
	m.robotPowerBusA.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainA))
	m.robotPowerBusB.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainB))
	m.robotWatchdogOK.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.WatchdogOK))
}

// MQTTInit connects to the mqtt server and listens for metrics.
func (m *Metrics) MQTTInit(wg *sync.WaitGroup) error {
	wg.Add(1)
	opts := mqtt.NewClientOptions().
		AddBroker(m.broker).
		SetAutoReconnect(true).
		SetClientID("self-metrics").
		SetConnectRetry(true).
		SetConnectTimeout(time.Second).
		SetConnectRetryInterval(time.Second)
	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		m.l.Error("Error connecting to broker", "error", tok.Error())
		return tok.Error()
	}
	m.l.Info("Connected to broker")

	subFunc := func() error {
		if tok := client.Subscribe("robot/+/stats", 1, m.mqttCallback); tok.Wait() && tok.Error() != nil {
			m.l.Warn("Error subscribing to topic", "error", tok.Error())
			return tok.Error()
		}
		return nil
	}
	if err := backoff.Retry(subFunc, backoff.NewExponentialBackOff()); err != nil {
		m.l.Error("Permanent error encountered while subscribing", "error", err)
		return err
	}
	m.l.Info("Subscribed to topics")
	wg.Done()
	return nil

}

// StartFlusher clears the stats for robots every 10 seconds
// ensuring that robots don't stick around as zombies in the system if
// they've disconnected.
func (m *Metrics) StartFlusher() {
	flushTicker := time.NewTicker(time.Second * 10)

	go func() {
		for {
			select {
			case <-m.stopStatFlusher:
				flushTicker.Stop()
				return
			case <-flushTicker.C:
				m.ResetRobotMetrics()
			}
		}
	}()
}

// Shutdown signals the flusher that we wish to cease operations.
func (m *Metrics) Shutdown() {
	m.stopStatFlusher <- struct{}{}
}

func fCast(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
