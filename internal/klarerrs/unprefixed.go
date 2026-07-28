package klarerrs

const (
	_ Code = NoPrefix + iota
)

func (e *Error) handleUnprefixed() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""
	}
}
