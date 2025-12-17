package klon

import (
	"errors"
	"reflect"
)

func unmarshallDstError(rt reflect.Type) error {
	if rt == nil {
		return errors.New("klon: nil argument passed to Unmarshall")
	}
	typ := rt.String()
	if rt.Kind() != reflect.Pointer {
		return errors.New("klon: non-pointer passed to Unmarshall: " + typ)
	}
	return errors.New("klon: nil " + typ + "passed to Unmarshall")
}
