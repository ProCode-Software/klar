package errors

// Tests to check if every error has a message. If getting an error message panics, then it
// has a message because it is accessing an attribute that should exist.
import (
	"fmt"
	"strings"
	"testing"
)

type errorType struct {
	prefix ErrorCode
	err    func(ErrorCode) KlarError
}

var errorTypes = map[string]errorType{
	"SyntaxError": {
		SyntaxErrorPrefix, func(ec ErrorCode) KlarError { return ParseError{ErrorCode: ec} },
	},
	"TypeError": {
		TypeErrorPrefix, func(ec ErrorCode) KlarError { return TypeError{ErrorCode: ec} },
	},
	"ReferenceError": {
		ReferenceErrorPrefix, func(ec ErrorCode) KlarError { return ReferenceError{ErrorCode: ec} },
	},
	"Warning": {
		WarningPrefix, func(ec ErrorCode) KlarError { return Warning{ErrorCode: ec} },
	},
}

func runTest(name string, spec errorType, t *testing.T) {
	for code := spec.prefix + 1; !strings.HasPrefix(code.String(), "ErrorCode("); code++ {
		func() {
			defer func() {
				if err := recover(); err != nil {
					return
				}
			}()
			err := spec.err(code)
			msg := err.Error()
			if msg == "" || strings.HasPrefix(msg, fmt.Sprintf("%s: %s", name, code)) {
				t.Errorf("missing: %s / %s", name, code)
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
