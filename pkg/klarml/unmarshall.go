package klarml

import (
	"errors"
	"fmt"
	"reflect"
)



type Unmarshaller interface {
	UnmarshalKlarMarkup(Node) error
}

func tryUnmarshaller(rv reflect.Value, data Node) (ok bool, err error) {
	if !rv.Elem().CanInterface() {
		return false, nil
	}
	if u, ok := rv.Interface().(Unmarshaller); ok {
		return true, u.UnmarshalKlarMarkup(data)
	}
	return false, nil
}

func UnmarshallDocument(doc Document, dst any) error {
	panic(errors.New("not implemented"))
	/* const prefix = "klarml.Unmarshall: "
	rv := reflect.ValueOf(dst)
	rt := reflect.TypeOf(dst)
	switch {
	case rv.Kind() != reflect.Pointer:
		return errors.New("klarml.Unmarshall(data, dst): dst must be a pointer")
	case rv.IsNil():
		return errors.New("klarml.Unmarshall(data, dst): dst must not be nil")
	}
	if !rv.IsValid() {
		return fmt.Errorf("klarml.Unmarshall: %v is not a valid reflect.Value", rv)
	}
	ctx := NewContext(doc)
	errors := ctx.ResolveVars()
	if len(errors) > 0 {
		return fmt.Errorf("klarml.Unmarshall: %w", errors[0])
	}
	ok, err := tryUnmarshaller(rv, doc.Body)
	if ok && err != nil {
		return err
	}
	elem := rv.Elem()
	typeElem := rt.Elem()
	switch node := doc.Body.(type) {
	case Object:
		
	case StringLiteral:
		if elem.Kind() != reflect.String {
			return fmt.Errorf(prefix+"cannot unmarshall string into non-string %T", rv)
		}
	case NumericLiteral:
		if !elem.CanFloat() {
			return fmt.Errorf(prefix+"cannot unmarshall number into non-numeric type %T", rv)
		}
	} */
	return nil
}

func Unmarshall(data []byte, dst any) error {
	doc, err := Parse(data)
	if len(err) > 0 {
		return fmt.Errorf("klarml.Unmarshall: parse error: %w", err[0])
	}
	return UnmarshallDocument(doc, dst)
}
