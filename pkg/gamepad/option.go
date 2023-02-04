package gamepad

import (
	"github.com/hashicorp/go-hclog"
)

// Option is used to enable variadic option passing to the joystick
// controller.
type Option func(jsc *JSController)

// WithLogger sets the logging instance for the gamepad controller.
func WithLogger(l hclog.Logger) Option {
	return func(jsc *JSController) {
		jsc.l = l.Named("gamepad-controller")
	}
}
