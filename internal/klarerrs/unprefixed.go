package klarerrs

const ErrTooManyErrors Code = -1

const (
	_ Code = NoPrefix + iota
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
