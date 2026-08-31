package klarerrs

const (
	_ Code = ImplementationErrorPrefix + iota

	ErrMissingImpl       // Missing implementation for some targets
	ErrUnsupportedTarget // Object isn't supported for specific targets
)

func (e *Error) handleImplementationError() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""
	}
}
