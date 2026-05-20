package klarerrs

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

//go:generate stringer -type=Code

type Error struct {
	Code Code
	File string

	// Positions
	Name  string
	Range ranges.Range
	Node  ast.Node
	Info  Info

	Label      string      // The message displayed next to the main highlight
	Highlights []Highlight // Additional highlights to display
	Details    []Detail    // Additional files context to display
	Hints      []Hint      // Hints to display

	Params ErrorParams // Internal details about the error
}

type ErrorParams map[string]any

// Code represents a numeric identifier for an error.
type Code int

const (
	NoPrefix Code = 100 * iota
	SyntaxErrorPrefix
	_
	WarningPrefix
	TypeErrorPrefix
	ReferenceErrorPrefix
	ModuleErrorPrefix
	ImplementationErrorPrefix
)

// Prefix returns the prefix of the error code.
func (e *Error) Prefix() Code {
	prefix := e.Code / 100
	if prefix == SyntaxErrorPrefix+1 {
		return SyntaxErrorPrefix
	}
	return prefix * 100
}

// Highlights are displayed below a certain range in a file alongside
// the main highlight in error messages.
type Highlight struct {
	Range   ranges.Range
	Message string
}

// Details show a separate view of a file below the main error source code.
// They may be labelled with a title
type Detail struct {
	File    string
	Range   ranges.Range
	Message string
}

// Hints are shown below source code in error messages, that display
// a message and optionally a diff.
type Hint struct {
	Message string
	Diff    *Diff
}

// Hint attaches a new hint to the error, returning the created [Hint].
func (e *Error) Hint(s string) *Hint {
	h := Hint{Message: s}
	e.Hints = append(e.Hints, h)
	return &h
}

func (e *Error) Hintf(format string, a ...any) *Hint {
	return e.Hint(fmt.Sprintf(format, a...))
}

func (e *Error) AddHighlight(msg string, r ranges.Range) *Highlight {
	h := Highlight{Message: msg, Range: r}
	e.Highlights = append(e.Highlights, h)
	return &h
}

func (e *Error) AddDetail(msg string, file string, r ranges.Range) *Detail {
	d := Detail{File: file, Range: r, Message: msg}
	e.Details = append(e.Details, d)
	return &d
}

// Message returns the error message.
func (e *Error) Message() string {
	switch e.Prefix() {
	case SyntaxErrorPrefix:
		return e.handleSyntaxError()
	case TypeErrorPrefix:
		return e.handleTypeError()
	case WarningPrefix:
		return e.handleWarning()
	case ReferenceErrorPrefix:
		return e.handleReferenceError()
	case NoPrefix:
		return e.handleUnprefixed()
	case ModuleErrorPrefix:
		return e.handleModuleError()
	case ImplementationErrorPrefix:
		// TODO: implementation errors
	default:
		panic(fmt.Sprintf("unhandled error prefix %d", e.Prefix()))
	}
	return ""
}

// Error implements [error] and is synonymous with [Error.Message].
func (e *Error) Error() string {
	return e.Message()
}

// Params
// ========

func (e *Error) SetParam(key string, val any) {
	if e.Params == nil {
		e.Params = make(ErrorParams)
	}
	e.Params[key] = val
}

func (e *Error) GetParam(key string) any {
	if e.Params == nil {
		return nil
	}
	return e.Params[key]
}

func (e *Error) StringParam(key string) string {
	if v := e.GetParam(key); v != nil {
		return v.(string)
	}
	return ""
}

func (e *Error) IntParam(key string) int {
	if v := e.GetParam(key); v != nil {
		return v.(int)
	}
	return 0
}

func (e *Error) BoolParam(key string) bool {
	if v := e.GetParam(key); v != nil {
		return v.(bool)
	}
	return false
}

func (e *Error) TokenTypeParam(key string) lexer.TokenType {
	if v := e.GetParam(key); v != nil {
		return v.(lexer.TokenType)
	}
	return 0
}

func (e *Error) noMessage() {
	panic(fmt.Sprintf("error %s doesn't have a message", e.Code))
}
