// Package klarerror provides references and helpers for errors used
// by the Klar compiler and runtime.
package klarerrors

import "github.com/ProCode-Software/klar/internal/klarerrs"

// IsCompileError returns true if err is not nil and any Klar compiler error.
func IsCompileError(err error) bool {
	_, ok := err.(*klarerrs.Error)
	return ok
}

// *Error is any error from the Klar compiler
type *Error = *klarerrs.Error

type (
	// A Warning is a compile-time warning
	Warning = klarerrs.Warning
	// SyntaxError is an alias for [klarerrs.Error]
	SyntaxError = klarerrs.Error
	// A Error is a syntax error during parsing or analysis
	Error = klarerrs.Error
	// A TypeError is a type error during analysis
	TypeError = klarerrs.TypeError
	// A ReferenceError is an error caused by an unknown or invalid reference during analysis
	ReferenceError = klarerrs.ReferenceError
)
