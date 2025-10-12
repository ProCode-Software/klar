package errors

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

var ErrMaxDepth = errors.New("klarml: maximum depth exceeded")

type InvalidUnmarshall struct {
	Type reflect.Type
}

type TypeError struct {
	Expected reflect.Type
	Value    ast.Node
}

type NumberRange struct {
	Value     float64
	Expected  reflect.Type
	Truncated bool // true: is supposed to be int; false: is supposed to be uint
}

type ExpectedToken struct {
	Expected, Got byte
}
type UnexpectedEOF struct {
	Expected byte
}

type ExpectedEOF struct {
	Got byte
}
type UnsupportedType struct {
	Type reflect.Type
}
type InvalidArrayLength struct {
	Need, Got int
}

func (err *InvalidUnmarshall) Error() string {
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

func (err *NumberRange) Error() string {
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

func (err *ExpectedEOF) Error() string {
	return fmt.Sprintf("klarml: expected end of file, but found %q instead", err.Got)
}

func (err *ExpectedToken) Error() string {
	expected := "'" + string(err.Expected) + "'"
	if err.Expected == '\n' {
		expected = "newline"
	}
	return fmt.Sprintf("klarml: expected %s, but found %q instead",
		expected, err.Got,
	)
}

func (err *UnsupportedType) Error() string {
	return fmt.Sprintf("klarml: unsupported Go type %s", err.Type.String())
}

func (err *InvalidArrayLength) Error() string {
	return fmt.Sprintf(
		"klarml: mismatched array length: expected %d items, but source has %d",
		err.Need, err.Got,
	)
}

func (err *UnexpectedEOF) Error() string {
	return fmt.Sprintf("klarml: expected %q, but found end of file", err.Expected)
}
