package klarerrs

// Tests to check if every error has a message. If getting an error message panics, then it
// has a message because it is accessing an attribute that should exist.
import (
	"fmt"
	"strings"
	"testing"
)

var errorTypes = map[string] Code{
	"SyntaxError": SyntaxErrorPrefix,
	"TypeError": TypeErrorPrefix,
	"ReferenceError": ReferenceErrorPrefix,
	"Warning": WarningPrefix,
}

func runTest(name string, prefix Code, t *testing.T) {
	for code := prefix + 1; !strings.HasPrefix(code.String(), "Code("); code++ {
		func() {
			defer func() {
				if err := recover(); err != nil {
					// If it panics, that means the error message needs data
					// (such as an AST node) to print the message, and therefore,
					// an error exists.
					return 
				}
			}()
			err := &Error{Code: code}
			msg := err.Error()
			if msg == "" || strings.HasPrefix(msg, fmt.Sprintf("%s: %s", name, code)) {
				t.Errorf("missing: %s - %s", name, code)
			}
		}()
	}
}

func TestErrorMessages(t *testing.T) {
	for name, spec := range errorTypes {
		t.Run(name, func(t *testing.T) {
			runTest(name, spec, t)
		})
	}
}
