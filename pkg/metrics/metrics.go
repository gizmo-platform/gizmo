package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// New returns an initialized instance of the metrics system.
func New(opts ...Option) *Metrics {
	x := &Metrics{
		l:               hclog.NewNullLogger(),
		r:               prometheus.NewRegistry(),
		broker:          "mqtt://127.0.0.1:1883",
		stopStatFlusher: make(chan (struct{})),
		lastSeen:        &sync.Map{},

		robotRSSI: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "rssi",
			Help:      "WiFi signal strength as measured by the system processor.",
		}, []string{"team"}),

		robotWifiReconnects: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "wifi_reconnects",
			Help:      "Number of reconnects since last boot",
		}, []string{"team"}),

		robotVBat: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "battery_voltage",
			Help:      "Robot Battery volage.",
		}, []string{"team"}),

		robotPowerBoard: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_board",
			Help:      "General logic power available.",
		}, []string{"team"}),

		robotPowerPico: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_pico",
			Help:      "Pico power supply available.",
		}, []string{"team"}),

		robotPowerGPIO: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_gpio",
			Help:      "GPIO power supply available.",
		}, []string{"team"}),

		robotPowerServo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_servo",
			Help:      "Servo power available.",
		}, []string{"team"}),

		robotPowerBusA: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_bus_a",
			Help:      "Motor Bus A power available.",
		}, []string{"team"}),

		robotPowerBusB: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_bus_b",
			Help:      "Motor Bus B power available.",
		}, []string{"team"}),

		robotPowerPixels: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "power_pixels",
			Help:      "Student Pixel power available.",
		}, []string{"team"}),

		robotWatchdogOK: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "watchdog_ok",
			Help:      "Watchdog has been fed and is alive.",
		}, []string{"team"}),

		robotWatchdogLifetime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "watchdog_remaining_seconds",
			Help:      "Watchdog lifetime remaining since last feed.",
		}, []string{"team"}),

		robotControlFrames: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "control_frames",
			Help:      "Count of control frames received since power on.",
		}, []string{"team"}),

		robotControlFrameAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "control_frame_age_seconds",
			Help:      "Time since last control frame was received",
		}, []string{"team"}),

		robotLastInteraction: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "gizmo",
			Subsystem: "robot",
			Name:      "last_interaction",
			Help:      "Timestamp of the last mqtt metrics push",
		}, []string{"team"}),
	}

	x.r.MustRegister(x.robotRSSI)
	x.r.MustRegister(x.robotWifiReconnects)
	x.r.MustRegister(x.robotVBat)
	x.r.MustRegister(x.robotPowerBoard)
	x.r.MustRegister(x.robotPowerPico)
	x.r.MustRegister(x.robotPowerGPIO)
	x.r.MustRegister(x.robotPowerServo)
	x.r.MustRegister(x.robotPowerBusA)
	x.r.MustRegister(x.robotPowerBusB)
	x.r.MustRegister(x.robotPowerPixels)
	x.r.MustRegister(x.robotWatchdogOK)
	x.r.MustRegister(x.robotWatchdogLifetime)
	x.r.MustRegister(x.robotControlFrames)
	x.r.MustRegister(x.robotControlFrameAge)
	x.r.MustRegister(x.robotLastInteraction)

	x.s = &http.Server{}

	for _, o := range opts {
		o(x)
	}

	return x
}

// BuiltinWebserver runs the metrics webserver when nothing else does.
func (m *Metrics) BuiltinWebserver(bind string) error {
	m.s.Addr = bind
	mux := &http.ServeMux{}
	mux.Handle("/metrics", promhttp.HandlerFor(m.r, promhttp.HandlerOpts{Registry: m.r}))
	m.s.Handler = mux
	go func() {
		<-m.stopStatFlusher
		m.s.Shutdown(context.Background())
	}()

	return m.s.ListenAndServe()
}

// Registry provides access to the registry that this instance
// manages.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.r
}

// DeleteZombieRobot removes metrics associated with a zombie robot
// that is no longer connected.
func (m *Metrics) DeleteZombieRobot(team string) {
	l := prometheus.Labels{"team": team}

	m.robotRSSI.Delete(l)
	m.robotWifiReconnects.Delete(l)
	m.robotVBat.Delete(l)
	m.robotPowerBoard.Delete(l)
	m.robotPowerPico.Delete(l)
	m.robotPowerGPIO.Delete(l)
	m.robotPowerServo.Delete(l)
	m.robotPowerBusA.Delete(l)
	m.robotPowerBusB.Delete(l)
	m.robotPowerPixels.Delete(l)
	m.robotWatchdogOK.Delete(l)
	m.robotWatchdogLifetime.Delete(l)
	m.robotControlFrameAge.Delete(l)
	m.robotControlFrames.Delete(l)
}

// MQTTCallback is called by external callers to process packets.
func (m *Metrics) MQTTCallback(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	teamNum := strings.Split(pk.TopicName, "/")[1]
	m.l.Trace("Called back", "team", teamNum)
	m.ParseReport(teamNum, pk.Payload)
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
				now := time.Now()
				m.lastSeen.Range(func(team, seen any) bool {
					if now.Sub(seen.(time.Time)) > time.Second*10 {
						m.DeleteZombieRobot(team.(string))
					}
					return false
				})
			}
		}
	}()
}

// Shutdown signals the flusher that we wish to cease operations.
func (m *Metrics) Shutdown() {
	m.stopStatFlusher <- struct{}{}
}

// ParseReport directly parses a report from a buffer.
func (m *Metrics) ParseReport(teamNum string, data []byte) error {
	var stats report
	if err := json.Unmarshal(data, &stats); err != nil {
		m.l.Warn("Bad stats report", "team", teamNum, "error", err)
		return err
	}

	// This uses the same conversion that's used on the Gizmo to
	// drive the battery status LED, which is why it has to have
	// access to the values from the Gizmo itself.
	voltage := (float64(stats.VBatM)/100000)*float64(stats.VBat) + (float64(stats.VBatM) / 100000)

	m.robotRSSI.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.RSSI))
	m.robotWifiReconnects.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WifiReconnects))
	m.robotVBat.With(prometheus.Labels{"team": teamNum}).Set(voltage)
	m.robotWatchdogLifetime.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.WatchdogRemaining) / 1000)
	m.robotControlFrameAge.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.ControlFrameAge) / 1000)
	m.robotControlFrames.With(prometheus.Labels{"team": teamNum}).Set(float64(stats.ControlFramesReceived))

	m.robotPowerBoard.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrBoard))
	m.robotPowerPico.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrPico))
	m.robotPowerGPIO.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrGPIO))
	m.robotPowerServo.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrServo))
	m.robotPowerBusA.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainA))
	m.robotPowerBusB.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrMainB))
	m.robotPowerPixels.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.PwrPixels))
	m.robotWatchdogOK.With(prometheus.Labels{"team": teamNum}).Set(fCast(stats.WatchdogOK))

	m.robotLastInteraction.With(prometheus.Labels{"team": teamNum}).SetToCurrentTime()
	m.lastSeen.Store(teamNum, time.Now())
	return nil
}

func fCast(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
