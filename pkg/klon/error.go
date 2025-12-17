package klon

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

type ErrorCode int

type ReadError struct {
	Err error
}

func (err ReadError) Error() string {
	return "error while parsing: " + err.Err.Error()
}

func (err ReadError) Unwrap() error {
	return err.Err
}

type InvalidUnmarshallError struct {
	Type reflect.Type
}

func (err *InvalidUnmarshallError) Error() string {
	switch {
	case err.Type == nil:
		return "klon: nil argument passed to Unmarshall"
	case err.Type.Kind() != reflect.Pointer:
		return "klon: non-pointer type " + err.Type.String() + " passed to Unmarshall"
	}
	return "klon: nil " + err.Type.String() + " passed to Unmarshall"
}

const (
	_ ErrorCode = iota
	ErrUnterminatedString
	ErrExpectedEOF
	ErrExpectedVarInArrow
	ExpectedCurlyInVar
	ErrInvalidArrow
	ErrUnterminatedVar
	ErrInvalidIdentifier
	ErrMaxDepth
	ErrUnterminatedList
	ErrExpectedToken
	ErrUnexpectedToken
	ErrDashWithoutNewline
	ErrDashAtTopLevel
	ErrTypeMismatch
	ErrNegativeNumber
	ErrUnsupportedValue
	ErrTruncatedNumber
	ErrUnknownEscape
	ErrWrongArrayLength
	ErrUnmatchedBracket
	ErrExpectedValue
	ErrIllegalCharacter
	ErrVariableNotDefined
)

type ParseError struct {
	Code  ErrorCode
	Range ranges.Range
	Token Token
	Text  string
}

func (err *ParseError) Error() string {
	return "klon: syntax error " + printLoc(err.Range) + ": " + err.Text
}

type TypeError struct {
	Code ErrorCode
	Type reflect.Type
	Val  ast.Value
	Text string
}

func (err *TypeError) Error() string {
	return "klon: type error " + printLoc(err.Val.Pos()) + ": " + err.Text
}

func printLoc(r ranges.Range) string {
	return fmt.Sprintf("at line %d, column %d", r.Start.Line, r.Start.Col)
}
