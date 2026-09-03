package klarerrs

const (
	_ Code = ImplementationErrorPrefix + iota

	ErrMissingImpl       // Missing implementation for some targets
	ErrUnsupportedTarget // Object isn't supported for specific targets
	ErrReservedJSKeyword // Public object name can't be a JS keyword
)

func (e *Error) handleImplementationError() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case ErrReservedJSKeyword:
		return Quote(e.Name) + " is a reserved keyword in JavaScript and can't be used as a name"
	}
}
