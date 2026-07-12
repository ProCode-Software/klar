package klon

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
)

// ReadError is returned when an error occurs while reading from the input.
type ReadError struct{ Err error }

func (err ReadError) Error() string {
	return "error while parsing: " + err.Err.Error()
}

func (err ReadError) Unwrap() error { return err.Err }

type Error struct {
	Code  klonerrs.Code
	Range ranges.Range
	Text  string
	Token Token // For syntax errors
	// For type errors
	Type        reflect.Type
	Value       ast.Value
	isDecodeErr bool
	Warning     bool
}

func (err *Error) Error() string {
	kind := "syntax"
	if err.Type != nil {
		kind = "type"
	}
	return fmt.Sprintf("klon: %s error at %s: %s", kind, err.Range, err.Text)
}

func (e *Error) IsTypeError() bool { return e.isDecodeErr }

// Implements [reporter.Error]
// =======

func (err *Error) Title() string {
	if err.IsTypeError() {
		return "Error"
	}
	return "Syntax error"
}
func (err *Error) Location() ranges.Range { return err.Range }
func (err *Error) Message() string        { return err.Text }

func (err *Error) ErrorCode() string                     { return "" }
func (err *Error) IsWarning() bool                       { return err.Warning }
func (err *Error) FilePath() string                      { return "" }
func (err *Error) Description() string                   { return "" }
func (err *Error) MainHighlight() string                 { return "" }
func (err *Error) ErrorDetails() []klarerrs.Detail       { return nil }
func (err *Error) ErrorHighlights() []klarerrs.Highlight { return nil }
func (err *Error) ErrorHints() []klarerrs.Hint           { return nil }
