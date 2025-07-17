package klarml

import (
	"errors"
	"reflect"
)

func unmarshallDstError(rt reflect.Type) error {
	if rt == nil {
		return errors.New("klarml: nil argument passed to Unmarshall")
	}
	typ := rt.String()
	if rt.Kind() != reflect.Pointer {
		return errors.New("klarml: non-pointer passed to Unmarshall: " + typ)
	}
	return errors.New("klarml: nil " + typ + "passed to Unmarshall")
}
