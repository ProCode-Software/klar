package klarerrs

// Tests to check if every error has a message. If getting an error message panics, then it
// has a message because it is accessing an attribute that should exist.
import (
	"fmt"
	"strings"
	"testing"
	"text/tabwriter"
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
			missingCodes = append(missingCodes, e.Code.String())
			t.Fail()
		}
	}
	if len(missingCodes) > 0 {
		t.Errorf("%d missing codes:", len(missingCodes))
		tw := tabwriter.NewWriter(t.Output(), 10, 4, 2, ' ', 0)
		for i := 0; i < len(missingCodes); {
			row := missingCodes[i:min(i+4, len(missingCodes))]
			tw.Write([]byte(strings.Join(row, "\t")))
			tw.Write([]byte{'\n'})
			i += len(row)
		}
		tw.Flush()
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
