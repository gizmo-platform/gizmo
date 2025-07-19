package eventstream

import (
	"encoding/json"
)

// PublishError pushes an error out into the event stream.
func (es *EventStream) PublishError(err error) {
	e := EventError{
		Type:  EventTypeError,
		Error: err.Error(),
	}

	bytes, err := json.Marshal(e)
	if err != nil {
		es.l.Warn("Error marshaling error", "error", err)
		return
	}
	es.publish(bytes)
}

// PublishLogLine pushes a log message into the event stream.
func (es *EventStream) PublishLogLine(msg string) {
	e := EventLogLine{
		Type:    EventTypeLogLine,
		Message: msg,
	}

	bytes, err := json.Marshal(e)
	if err != nil {
		es.l.Warn("Error marshaling error", "error", err)
		return
	}
	es.publish(bytes)
}

// PublishActionStart pushes an asynchronous action start message.
func (es *EventStream) PublishActionStart(action, msg string) {
	e := EventActionStart{
		Type:    EventTypeActionStart,
		Action:  action,
		Message: msg,
	}

	bytes, err := json.Marshal(e)
	if err != nil {
		es.l.Warn("Error marshaling error", "error", err)
		return
	}
	es.publish(bytes)
}

// PublishFileFetch pushes a filename into the event stream.
func (es *EventStream) PublishFileFetch(file string) {
	e := EventFileFetch{
		Type:     EventTypeFileFetch,
		Filename: file,
	}

	bytes, err := json.Marshal(e)
	if err != nil {
		es.l.Warn("Error marshaling error", "error", err)
		return
	}
	es.publish(bytes)
}
