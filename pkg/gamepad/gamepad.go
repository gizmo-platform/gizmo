package gamepad

import (
	"errors"
	"sync"

	"github.com/0xcafed00d/joystick"
	"github.com/hashicorp/go-hclog"
)

var (
	// ErrNoSuchField is returned in the event that no field is
	// available for the given fieldID.
	ErrNoSuchField = errors.New("no field with that ID exists")
)

// Values abstracts over the joystick to the given values that
// are returned by a gamepad.
type Values struct {
	AxisLX           int
	AxisLY           int
	AxisRX           int
	AxisRY           int
	AxisLT           int
	AxisRT           int
	AxisDX           int
	AxisDY           int
	ButtonBack       bool
	ButtonStart      bool
	ButtonLogo       bool
	ButtonLeftStick  bool
	ButtonRightStick bool
	ButtonX          bool
	ButtonY          bool
	ButtonA          bool
	ButtonB          bool
	ButtonLShoulder  bool
	ButtonRShoulder  bool
}

// JSController handles the action of actually fetching data from the
// joystick and making it available to the rest of the system.
type JSController struct {
	l hclog.Logger

	controllers map[string]joystick.Joystick
	state       map[string]*Values

	cMutex sync.RWMutex
	sMutex sync.RWMutex
}

// NewJSController sets up the joystick controller.
func NewJSController(opts ...Option) JSController {
	jsc := JSController{
		l:           hclog.NewNullLogger(),
		controllers: make(map[string]joystick.Joystick),
		state:       make(map[string]*Values),
	}

	for _, o := range opts {
		o(&jsc)
	}
	return jsc
}

// BindController attaches a controller to a particular name.
func (j *JSController) BindController(name string, id int) error {
	js, jserr := joystick.Open(id)
	if jserr != nil {
		return jserr
	}
	j.controllers[name] = js
	j.l.Info("Successfully bound controller", "fid", name, "jsid", id)
	return nil
}

// GetState fetches the state for a single field quadrant.
func (j *JSController) GetState(fieldID string) (*Values, error) {
	j.sMutex.RLock()
	defer j.sMutex.RUnlock()

	val, ok := j.state[fieldID]
	if !ok {
		return nil, ErrNoSuchField
	}
	j.l.Trace("Provided state", "fid", fieldID)
	return val, nil
}

// UpdateState polls the joystick and updates the values available to
// the controller.
func (j *JSController) UpdateState(fieldID string) error {
	j.cMutex.RLock()
	defer j.cMutex.RUnlock()

	js, ok := j.controllers[fieldID]
	if !ok {
		return ErrNoSuchField
	}

	jinfo, err := js.Read()
	if err != nil {
		return err
	}
	jvals := Values{
		AxisLX: jinfo.AxisData[0],
		AxisLY: jinfo.AxisData[1],

		AxisRX: jinfo.AxisData[3],
		AxisRY: jinfo.AxisData[4],

		AxisLT: jinfo.AxisData[2],
		AxisRT: jinfo.AxisData[5],

		AxisDX: jinfo.AxisData[6],
		AxisDY: jinfo.AxisData[7],

		ButtonBack:       (jinfo.Buttons & (1 << uint32(6))) != 0,
		ButtonStart:      (jinfo.Buttons & (1 << uint32(7))) != 0,
		ButtonLogo:       (jinfo.Buttons & (1 << uint32(8))) != 0,
		ButtonLeftStick:  (jinfo.Buttons & (1 << uint32(9))) != 0,
		ButtonRightStick: (jinfo.Buttons & (1 << uint32(10))) != 0,
		ButtonX:          (jinfo.Buttons & (1 << uint32(2))) != 0,
		ButtonY:          (jinfo.Buttons & (1 << uint32(3))) != 0,
		ButtonA:          (jinfo.Buttons & (1 << uint32(0))) != 0,
		ButtonB:          (jinfo.Buttons & (1 << uint32(1))) != 0,
		ButtonLShoulder:  (jinfo.Buttons & (1 << uint32(4))) != 0,
		ButtonRShoulder:  (jinfo.Buttons & (1 << uint32(5))) != 0,
	}

	j.sMutex.Lock()
	j.state[fieldID] = &jvals
	j.sMutex.Unlock()
	j.l.Trace("Refreshed state", "fid", fieldID)
	return nil
}
