package metrics

import (
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics binds the registry as well as the metrics collection.
type Metrics struct {
	l      hclog.Logger
	broker string

	r *prometheus.Registry

	robotRSSI             *prometheus.GaugeVec
	robotWifiReconnects   *prometheus.GaugeVec
	robotVBat             *prometheus.GaugeVec
	robotPowerBoard       *prometheus.GaugeVec
	robotPowerPico        *prometheus.GaugeVec
	robotPowerGPIO        *prometheus.GaugeVec
	robotPowerServo       *prometheus.GaugeVec
	robotPowerBusA        *prometheus.GaugeVec
	robotPowerBusB        *prometheus.GaugeVec
	robotPowerPixels      *prometheus.GaugeVec
	robotWatchdogOK       *prometheus.GaugeVec
	robotWatchdogLifetime *prometheus.GaugeVec
	robotControlFrameAge  *prometheus.GaugeVec
	robotControlFrames    *prometheus.GaugeVec
	robotLastInteraction  *prometheus.GaugeVec

	robotOnField *prometheus.GaugeVec

	stopStatFlusher chan struct{}
	lastSeen        *sync.Map
}

type report struct {
	ControlFrameAge       int32
	ControlFramesReceived int32
	VBat                  int32
	VBatM                 int32
	VBatB                 int32
	WatchdogRemaining     int32
	WatchdogOK            bool
	RSSI                  int32
	WifiReconnects        int32
	PwrBoard              bool
	PwrPico               bool
	PwrGPIO               bool
	PwrServo              bool
	PwrMainA              bool
	PwrMainB              bool
	PwrPixels             bool
}

// Option provides a configuration framework to setup the metrics
// package.
type Option func(m *Metrics)
