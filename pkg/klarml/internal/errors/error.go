package errors

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

type InvalidUnmarshallError struct {
	Type reflect.Type
}

type TypeError struct {
	Expected reflect.Type
	Value    ast.Node
}

type NumberRangeError struct {
	Value     float64
	Expected  reflect.Type
	Truncated bool // true: is supposed to be int; false: is supposed to be uint
}

type ExpectedTokenError struct {
	Expected, Got byte
}

type ExpectedEOFError struct {
	Got byte
}

func (err *InvalidUnmarshallError) Error() string {
	switch {
	case err.Type == nil:
		return "klarml: nil argument passed to Unmarshall"
	case err.Type.Kind() != reflect.Pointer:
		return "klarml: non-pointer type " + err.Type.String() + " passed to Unmarshall"
	}
	return "klarml: nil " + err.Type.String() + " passed to Unmarshall"
}

func (err *TypeError) Error() string {
	return "klarml: type error"
}

func (err *NumberRangeError) Error() string {
	if err.Truncated {
		return fmt.Sprintf(
			"klarml: can't unmarshall float %f into Go integer type %s",
			err.Value, err.Expected.String(),
		)
	}
	return fmt.Sprintf(
		"klarml: can't unmarshall negative integer %d into Go unsigned type %s",
		int(err.Value), err.Expected.String(),
	)
}

func (err *ExpectedEOFError) Error() string {
	return fmt.Sprintf("klarml: expected end of file, but found '%c' instead", err.Got)
}

func (err *ExpectedTokenError) Error() string {
	return fmt.Sprintf("klarml: expected '%c', but found '%c' instead",
		err.Expected, err.Got,
	)
}
