package stats

import (
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics is a container for all prometheus metrics that are
// happening on this instance.
type Metrics struct {
	l hclog.Logger

	rssi              *prometheus.GaugeVec
	vbat              *prometheus.GaugeVec
	powerBoard        *prometheus.GaugeVec
	powerPico         *prometheus.GaugeVec
	powerGPIO         *prometheus.GaugeVec
	powerMainA        *prometheus.GaugeVec
	powerMainB        *prometheus.GaugeVec
	watchdogOK        *prometheus.GaugeVec
	watchdogRemaining *prometheus.GaugeVec
}

// NewListener hands back a complete stats listener for external
// consumption.
func NewListener(p hclog.Logger) (*prometheus.Registry, *Metrics) {
	reg := prometheus.NewRegistry()
	m := new(Metrics)
	m.l = p.Named("stats")
	m.rssi = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "rssi",
	}, []string{"team"})
	m.vbat = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "vbat",
	}, []string{"team"})
	m.powerBoard = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "powerBoard",
	}, []string{"team"})
	m.powerPico = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "powerPico",
	}, []string{"team"})
	m.powerGPIO = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "powerGPIO",
	}, []string{"team"})
	m.powerMainA = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "powerMainA",
	}, []string{"team"})
	m.powerMainB = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "powerMainB",
	}, []string{"team"})
	m.watchdogOK = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "watchdogOK",
	}, []string{"team"})
	m.watchdogRemaining = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "BEST",
		Subsystem: "robot",
		Name:      "watchdogRemaining",
	}, []string{"team"})
	reg.MustRegister(m.rssi)
	reg.MustRegister(m.vbat)
	reg.MustRegister(m.powerBoard)
	reg.MustRegister(m.powerPico)
	reg.MustRegister(m.powerGPIO)
	reg.MustRegister(m.powerMainA)
	reg.MustRegister(m.powerMainB)
	reg.MustRegister(m.watchdogOK)
	reg.MustRegister(m.watchdogRemaining)
	return reg, m
}
