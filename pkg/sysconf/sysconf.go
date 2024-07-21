package sysconf

import (
	"io/fs"

	"github.com/hashicorp/go-hclog"
)

// SysConf contains convenient functions for configuring a Void System
type SysConf struct {
	*Runit

	l hclog.Logger

	efs fs.FS
}

// Option configures the SysConf instance.
type Option func(*SysConf)

// New configures a SysConf instance and returns it.
func New(opts ...Option) *SysConf {
	sc := new(SysConf)
	sc.l = hclog.NewNullLogger()

	for _, o := range opts {
		o(sc)
	}

	return sc
}

// WithLogger sets the parent logger for the sysconf.
func WithLogger(l hclog.Logger) Option { return func(sc *SysConf) { sc.l = l.Named("sysconf") } }

// WithFS sets the filesystem that templates will be read out of.
func WithFS(f fs.FS) Option { return func(sc *SysConf) { sc.efs = f } }
