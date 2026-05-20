package klarerrs

const (
	ErrTooManyErrors Code = -1
)

func (e *Error) handleUnprefixed() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case ErrTooManyErrors:
		return "Too many errors"
	}
}
