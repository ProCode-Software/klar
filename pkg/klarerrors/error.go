// Package klarerror provides references and helpers for errors used
// by the Klar compiler and runtime.
package klarerrors

import "github.com/ProCode-Software/klar/internal/errors"

// IsCompileError returns true if err is not nil and any Klar compiler error.
func IsCompileError(err error) bool {
	_, ok := err.(errors.CompileError)
	return ok
}

// CompileError is any error from the Klar compiler
type CompileError = errors.CompileError

type (
	// A Warning is a compile-time warning
	Warning = errors.Warning
	// SyntaxError is an alias for [errors.ParseError]
	SyntaxError = errors.ParseError
	// A ParseError is a syntax error during parsing or analysis
	ParseError = errors.ParseError
	// A TypeError is a type error during analysis
	TypeError = errors.TypeError
	// A ReferenceError is an error caused by an unknown or invalid reference during analysis
	ReferenceError = errors.ReferenceError
)
