package errors

import (
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (err *ParseError) GetRange() ranges.Range     { return err.Range }
func (err *ParseError) Code() ErrorCode            { return err.ErrorCode }
func (err *ParseError) GetHints() []Hint           { return err.Hints }
func (err *ParseError) GetFile() string            { return err.File }
func (err *ParseError) GetDetails() []Detail       { return err.Details }
func (err *ParseError) GetLabel() string           { return err.Label }
func (err *ParseError) GetHighlights() []Highlight { return err.Highlights }
func (err *ParseError) Hint(hint string) {
	err.Hints = append(err.Hints, Hint{Message: hint})
}

func (err *ParseError) Hintf(hint string, a ...any) {
	err.Hints = hintf(err.Hints, hint, a)
}

func (err *TypeError) GetRange() ranges.Range     { return err.Range }
func (err *TypeError) Code() ErrorCode            { return err.ErrorCode }
func (err *TypeError) GetHints() []Hint           { return err.Hints }
func (err *TypeError) GetFile() string            { return err.File }
func (err *TypeError) GetDetails() []Detail       { return err.Details }
func (err *TypeError) GetLabel() string           { return err.Label }
func (err *TypeError) GetHighlights() []Highlight { return err.Highlights }
func (err *TypeError) Hint(hint string) {
	err.Hints = append(err.Hints, Hint{Message: hint})
}

func (err *TypeError) Hintf(hint string, a ...any) {
	err.Hints = hintf(err.Hints, hint, a)
}

func (err *Warning) GetRange() ranges.Range     { return err.Range }
func (err *Warning) Code() ErrorCode            { return err.ErrorCode }
func (err *Warning) GetHints() []Hint           { return err.Hints }
func (err *Warning) GetFile() string            { return err.File }
func (err *Warning) GetDetails() []Detail       { return err.Details }
func (err *Warning) GetLabel() string           { return err.Label }
func (err *Warning) GetHighlights() []Highlight { return err.Highlights }
func (err *Warning) Hint(hint string) {
	err.Hints = append(err.Hints, Hint{Message: hint})
}

func (err *Warning) Hintf(hint string, a ...any) {
	err.Hints = hintf(err.Hints, hint, a)
}

func (err *ReferenceError) GetRange() ranges.Range     { return err.Range }
func (err *ReferenceError) Code() ErrorCode            { return err.ErrorCode }
func (err *ReferenceError) GetHints() []Hint           { return err.Hints }
func (err *ReferenceError) GetFile() string            { return err.File }
func (err *ReferenceError) GetDetails() []Detail       { return err.Details }
func (err *ReferenceError) GetLabel() string           { return err.Label }
func (err *ReferenceError) GetHighlights() []Highlight { return err.Highlights }
func (err *ReferenceError) Hint(hint string) {
	err.Hints = append(err.Hints, Hint{Message: hint})
}

func (err *ReferenceError) Hintf(hint string, a ...any) {
	err.Hints = hintf(err.Hints, hint, a)
}
