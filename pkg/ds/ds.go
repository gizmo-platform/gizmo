package ds

import (
	"github.com/hashicorp/go-hclog"
)

// New returns a configured driverstation.
func New(opts ...Option) *DriverStation {
	d := new(DriverStation)
	d.l = hclog.NewNullLogger()
	d.svc = new(Runit)

	for _, o := range opts {
		o(d)
	}
	return d
}
