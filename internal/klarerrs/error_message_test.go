package klarerrs

// Tests to check if every error has a message. If getting an error message panics, then it
// has a message because it is accessing an attribute that should exist.
import (
	"fmt"
	"strings"
	"testing"
)

func TestErrorMessages(t *testing.T) {
	missingCodes := []string{}
	for code := Code(1); ; code++ {
		if code%100 == 0 && code != SyntaxErrorPrefix+100 {
			continue // x00
		}
		ok := true
		e := &Error{Code: code}
		if noTitle(e) {
			break
		}
		if strings.HasPrefix(code.String(), "Code(") {
			// Go to next prefix
			code = (code/100 + 1) * 100 // x00 is skipped by 'continue'
			continue
		}
		func() {
			defer func() {
				if r, _ := recover().(string); r != "" {
					if _, err := fmt.Sscanf(
						r, "error %s doesn't have a message", new(string),
					); err == nil {
						ok = false
					}
				}
			}()
			msg := e.Error()
			if msg == "" || strings.HasPrefix(msg, fmt.Sprintf("%s: %s", e.Title(), code)) {
				ok = false
			}
		}()
		if !ok {
			missingCodes = append(missingCodes, fmt.Sprintf("%s/%s", e.Title(), e.Code))
			t.Fail()
		}
	}
	if len(missingCodes) > 0 {
		var b strings.Builder
		for i := 0; i < len(missingCodes); i += 3 {
			line := missingCodes[i:min(i+3, len(missingCodes))]
			b.WriteString(strings.Join(line, "    "))
			b.WriteByte('\n')
		}
		t.Errorf("%d missing codes:\n\n%s", len(missingCodes), b.String())
	}
}

func noTitle(e *Error) (noTitle bool) {
	defer func() {
		if r := recover(); r != nil {
			noTitle = true
		}
	}()
	noTitle = e.Title() == ""
	return
}
