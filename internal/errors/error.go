package errors

type KlarError interface {
	error
	KlarError()
}

func (ParseError) KlarError() {}
