package eventstream

// NullStream doesn't publish events anywhere and is mostly for
// testing or non-server CLI cmdlets.
type NullStream struct{}

// NewNullStreamer hands back a null stream instance that discards
// everything.
func NewNullStreamer() *NullStream {
	return new(NullStream)
}

// PublishError discards all errors.
func (ns *NullStream) PublishError(_ error) {}

// PublishLogLine discards all log lines.
func (ns *NullStream) PublishLogLine(_ string) {}

// PublishActionStart discards all actions.
func (ns *NullStream) PublishActionStart(_, _ string) {}

// PublishFileFetch discards all filenames.
func (ns *NullStream) PublishFileFetch(_ string) {}
