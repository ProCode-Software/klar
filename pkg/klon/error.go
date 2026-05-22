package klon

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

// ReadError is returned when an error occurs while reading from the input.
type ReadError struct{ Err error }

func (err ReadError) Error() string {
	return "error while parsing: " + err.Err.Error()
}

func (err ReadError) Unwrap() error { return err.Err }

// InvalidUnmarshallError is returned when the target type is not a pointer or is nil.
type InvalidUnmarshallError struct{ Type reflect.Type }

func (err *InvalidUnmarshallError) Error() string {
	switch {
	case err.Type == nil:
		return "klon: nil argument passed to Unmarshall"
	case err.Type.Kind() != reflect.Pointer:
		return "klon: non-pointer type " + err.Type.String() + " passed to Unmarshall"
	}
	return "klon: nil " + err.Type.String() + " passed to Unmarshall"
}

type Code int

const (
	_ Code = iota

	ErrUnexpectedToken // Token not supposed to be there
	ErrExpectedToken   // Expected kind of token but got different type

	// Punctuation =====

	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedList    // A list that was left open
	ErrUnterminatedObject  // An object that was left open
	ErrUnterminatedVar     // A variable reference that was left open
	ErrUnterminatedComment // A block comment that was left open
	ErrExpectedCurlyInVar  // Missing '{' in variable reference
	ErrUnmatchedBracket    // Closing bracket without an opening one
	ErrIllegalCharacter    // Invalid Unicode character
	ErrExpectedEOF         // Content found after what should be the end of the document

	// Literal =====

	ErrInvalidIdentifier // Variable name starting with a digit
	ErrNegativeNumber    // Unexpected negative number
	ErrTruncatedNumber   // Float value where an integer was expected
	ErrUnknownEscape     // Invalid backslash escape sequence
	ErrInvalidKey        // Non-string/number/bool used as a key
	ErrExpectedValue     // Expected a value but found something else
	ErrExpectedClassName // Missing or invalid class name after '@'

	// Structural =====

	ErrDashWithoutNewline // Dash for nesting not preceded by a newline
	ErrDashAtTopLevel     // Dash found at the beginning of the document
	ErrDashSkip           // Dash depth increased by more than 1
	ErrMaxDepth           // Nesting depth exceeded MaxDepth

	// Variables =====

	ErrUndefinedVar       // Use of an undefined variable
	ErrVarNotTopLevel     // Variable declaration outside of top level
	ErrInvalidVarDecl     // Variable declaration using braces
	ErrExpectedVarInArrow // Missing variable after '<-'
	ErrInvalidArrow       // Rest/spread operator used in invalid context

	// Decode =====

	ErrTypeMismatch     // Mixed keyed and unkeyed entries in a block
	ErrWrongArrayLength // Mismatched array length during decoding
	ErrUnsupportedValue // Value type that cannot be decoded into the target Go type
)

type Error struct {
	Code  Code
	Range ranges.Range
	Token Token
	Text  string
}

func (err *Error) Error() string {
	return "klon: syntax error " + printLoc(err.Range) + ": " + err.Text
}

type TypeError struct {
	Code Code
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
