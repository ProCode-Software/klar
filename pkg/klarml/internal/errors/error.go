package errors

import "reflect"

type ErrorCode int

type InvalidUnmarshallError struct {
	Type reflect.Type
}

func (err *InvalidUnmarshallError) Error() string {
	return ""
}
