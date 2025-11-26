package errors

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

var ErrMaxDepth = errors.New("klarml: maximum depth exceeded")

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
type UnexpectedEOFError struct {
	Expected byte
}

type ExpectedEOFError struct {
	Got byte
}
type UnsupportedTypeError struct {
	Type reflect.Type
}
type InvalidArrayLengthError struct {
	Need, Got int
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
	return fmt.Sprintf("klarml: can't serialize %s into Go %s type",
		reflect.TypeOf(err.Value).Elem().Name(),
		err.Expected.String(),
	)
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
	return fmt.Sprintf("klarml: expected end of file, but found %q instead", err.Got)
}

func (err *ExpectedTokenError) Error() string {
	expected := "'" + string(err.Expected) + "'"
	if err.Expected == '\n' {
		expected = "newline"
	}
	return fmt.Sprintf("klarml: expected %s, but found %q instead",
		expected, err.Got,
	)
}

func (err *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("klarml: unsupported Go type %s", err.Type.String())
}

func (err *InvalidArrayLengthError) Error() string {
	return fmt.Sprintf(
		"klarml: mismatched array length: expected %d items, but source has %d",
		err.Need, err.Got,
	)
}

func (err *UnexpectedEOFError) Error() string {
	return fmt.Sprintf("klarml: expected %q, but found end of file", err.Expected)
}
