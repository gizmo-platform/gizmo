package watchdog

import (
	"time"

	"github.com/hashicorp/go-hclog"
)

// Option changes features on the dog.
type Option func(*Dog)

// The DogHandFunc is the hand that the dog bites if it doesn't get
// fed frequently enough.
type DogHandFunc func()

// Dog handles the time since its last fed, and the callback that will
// happen if the Dog decides to bite people.
type Dog struct {
	l hclog.Logger

	name string
	t    *time.Timer

	biteFunc     DogHandFunc
	foodDuration time.Duration
}

// New gets you a new watchdog.
func New(opts ...Option) *Dog {
	d := &Dog{
		name: "spot",
		l:    hclog.NewNullLogger(),

		biteFunc:     func() {},
		foodDuration: time.Second * 10,
	}
	for _, o := range opts {
		o(d)
	}
	d.t = time.AfterFunc(d.foodDuration, d.Bite)
	return d
}

// Bite calls the BiteFunction if nothing has called Feed within the
// specified number of leeway settings.
func (d *Dog) Bite() {
	d.l.Error("BITE!", "dog", d.name)
	d.t.Stop()
	d.biteFunc()
}

// Feed convinces the dog not to bite for the values specified during
// initialization, by default another 10 seconds.
func (d *Dog) Feed() {
	d.t.Reset(d.foodDuration)
}

// WithHandFunction sets up the hand that the dog will bite.  Not
// setting this kind of defeats the point of having a watchdog.
func WithHandFunction(f DogHandFunc) Option { return func(d *Dog) { d.biteFunc = f } }

// WithFoodDuration sets up how long the dog stays fed for when you
// call Feed().
func WithFoodDuration(fd time.Duration) Option { return func(d *Dog) { d.foodDuration = fd } }

// WithName names the dog.  If you don't specify this, you'll likely
// get bit by a dog named spot.
func WithName(n string) Option { return func(d *Dog) { d.name = n } }

// WithLogger provides a logging instance to the watchdog, since you
// probably do not want a silent dog wandering around biting
// goroutines.
func WithLogger(l hclog.Logger) Option { return func(d *Dog) { d.l = l.Named("watchdog") } }
