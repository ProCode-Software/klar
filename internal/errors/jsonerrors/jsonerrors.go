// Package jsonerrors provides functions for printing build results and errors in JSON format.
package jsonerrors

import (
	"encoding/json/v2"
	"io"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/errors"
)

// WriteTo writes the build result and error information to w in JSON format.
func WriteTo(w io.Writer, res *build.BuildResult, fatalErr error, isMaxErrors bool) error {
	// TODO: add error params
	format := struct {
		ElapsedTime   time.Duration `json:"elapsedTime,format:units"`
		ErrorCount    int           `json:"errorCount"`
		TooManyErrors bool          `json:"tooManyErrors,omitempty,omitzero"`
		Errors        errorSlice    `json:"errors"`
		FatalError    error         `json:"fatalError,omitempty,omitzero"`
	}{
		ElapsedTime:   res.Elapsed,
		TooManyErrors: isMaxErrors,
		ErrorCount:    len(res.Errors),
		Errors:        errorSlice(res.Errors),
		FatalError:    fatalErr,
	}
	return json.MarshalWrite(w, format, json.DefaultOptionsV2())
}

type errorSlice []errors.CompileError

func (es errorSlice) MarshalJSON() ([]byte, error) {
	errs := make([]compileError, len(es))
	for i, err := range es {
		r := err.GetRange()
		errs[i] = compileError{
			Message:    err.GetMessage(),
			Range:      rang{pos(r.Start), pos(r.End)},
			File:       err.GetFile(),
			Type:       err.GetName(),
			Code:       convertCode(err.GetCode()),
			Label:      err.GetLabel(),
			Hints:      convertHints(err.GetHints()),
			Details:    convertDetails(err.GetDetails()),
			Highlights: convertHighlights(err.GetHighlights()),
		}
	}
	return json.Marshal(errs, json.DefaultOptionsV2())
}

type compileError struct {
	Type       string      `json:"type"`
	Code       code        `json:"code"`
	Message    string      `json:"message"`
	File       string      `json:"file"`
	Range      rang        `json:"range"`
	Label      string      `json:"label,omitempty"`
	Hints      []hint      `json:"hints,omitempty"`
	Details    []detail    `json:"details,omitempty"`
	Highlights []highlight `json:"highlights,omitempty"`
}

type pos struct {
	Line uint32 `json:"line"`
	Col  uint32 `json:"column"`
}
type rang struct {
	Start pos `json:"start"`
	End   pos `json:"end"`
}
type hint struct {
	Message string `json:"message"`
}
type detail struct {
	File string `json:"file"`
	highlight
}
type highlight struct {
	Range   rang   `json:"range"`
	Message string `json:"message"`
}
type code struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

func convertCode(cd errors.ErrorCode) code {
	return code{Name: cd.Format(), ID: int(cd)}
}

func convertHints(hints []errors.Hint) []hint {
	hs := make([]hint, len(hints))
	for i, hn := range hints {
		hs[i] = hint{hn.Message}
	}
	return hs
}

func convertDetails(details []errors.Detail) []detail {
	ds := make([]detail, len(details))
	for i, dt := range details {
		ds[i] = detail{
			File: dt.File,
			highlight: highlight{
				Range:   rang{pos(dt.Range.Start), pos(dt.Range.End)},
				Message: dt.Message,
			},
		}
	}
	return ds
}

func convertHighlights(highlights []errors.Highlight) []highlight {
	hs := make([]highlight, len(highlights))
	for i, ht := range highlights {
		hs[i] = highlight{
			Range:   rang{pos(ht.Range.Start), pos(ht.Range.End)},
			Message: ht.Message,
		}
	}
	return hs
}
