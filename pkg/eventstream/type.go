package eventstream

// EventType is used to identify what type of event is crossing the
// wire.
type EventType uint8

const (
	// EventTypeUnknown is used as a zero value to ensure that this
	// always has to be set to something.
	EventTypeUnknown EventType = iota

	// EventTypeError is pushed across the wire in the event that the
	// system has encountered some kind of error that was
	// non-recoverable.
	EventTypeError

	// EventTypeLogLine is used to signify that the event in question
	// is a log line, which may be associated with one or more
	// other events.
	EventTypeLogLine

	// EventTypeActionStart is used to signal that an asynchronous
	// action has been requested, and that the system is
	// attempting to process it.
	EventTypeActionStart

	// EventTypeFileFetch is fired when a file is successfully
	// retrieved from a remote source.
	EventTypeFileFetch
)

// EventError contains the underlying error that occured.
type EventError struct {
	Type  EventType
	Error string
}

// EventLogLine contains a message from a log.
type EventLogLine struct {
	Type    EventType
	Message string
}

// EventActionStart contains an action name, and a message
type EventActionStart struct {
	Type    EventType
	Action  string
	Message string
}

// EventFileFetch contains a filename that was fetched successfully.
type EventFileFetch struct {
	Type     EventType
	Filename string
}
