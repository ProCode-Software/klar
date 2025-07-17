package errors

import "reflect"

type InvalidUnmarshallError struct {
	Type reflect.Type
}

func (err *InvalidUnmarshallError) Error() string {
	return ""
}