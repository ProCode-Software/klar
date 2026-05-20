// Package klarerror provides references and helpers for errors used
// by the Klar compiler and runtime.
package klarerrors

import "github.com/ProCode-Software/klar/internal/klarerrs"

// IsCompileError returns true if err is not nil and any Klar compiler error.
func IsCompileError(err error) bool {
	_, ok := err.(*klarerrs.Error)
	return ok
}

// Error is any error from the Klar compiler
type Error = *klarerrs.Error
