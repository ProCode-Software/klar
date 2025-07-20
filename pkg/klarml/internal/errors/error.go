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
	return "klarml: can't unmarshall"
}

func (err *TypeError) Error() string {
	return "klarml: type error"
}

func (err *NumberRangeError) Error() string {
	return "klarml: number range error"
}
func (err *ExpectedEOFError) Error() string {
	return fmt.Sprintf("klarml: expected end of file, but found '%c' instead", err.Got)
}

func (err *ExpectedTokenError) Error() string {
	return fmt.Sprintf("klarml: expected '%c', but found '%c' instead",
		err.Expected, err.Got,
	)
}
