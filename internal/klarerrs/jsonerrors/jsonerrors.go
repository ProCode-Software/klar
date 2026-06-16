// Package jsonerrors provides functions for printing build results and errors in JSON format.
package jsonerrors

import (
	"encoding/json/v2"
	"io"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// WriteTo writes the build result and error information to w in JSON format.
func WriteTo(w io.Writer, res *build.Result, fatalErr error, isMaxErrors bool) error {
	// TODO: add error params
	format := struct {
		ElapsedTime time.Duration `json:"elapsedTime,format:units"`
		ErrorCount  int           `json:"errorCount"`
		IsMaxErrors bool          `json:"maxErrors,omitempty,omitzero"`
		Errors      errorSlice    `json:"errors"`
		Warnings    errorSlice    `json:"warnings"`
		FatalError  error         `json:"fatalError,omitempty,omitzero"`
	}{
		ElapsedTime: res.Elapsed,
		IsMaxErrors: isMaxErrors,
		ErrorCount:  len(res.Errors),
		Errors:      res.Errors,
		Warnings:    res.Warnings,
		FatalError:  fatalErr,
	}
	return json.MarshalWrite(w, format, json.DefaultOptionsV2())
}

type errorSlice []*klarerrs.Error

func (es errorSlice) MarshalJSON() ([]byte, error) {
	errs := make([]compileError, len(es))
	for i, err := range es {
		errs[i] = compileError{
			Message:    err.Message(),
			Range:      rang{pos(err.Range.Start), pos(err.Range.End)},
			File:       err.File,
			Type:       err.Title(),
			Code:       convertCode(err.Code),
			Label:      err.Label,
			Hints:      convertHints(err.Hints),
			Details:    convertDetails(err.Details),
			Highlights: convertHighlights(err.Highlights),
			Info:       err.Info,
		}
	}
	return json.Marshal(errs, json.DefaultOptionsV2())
}

type compileError struct {
	Type       string        `json:"type"`
	Code       code          `json:"code"`
	Message    string        `json:"message"`
	File       string        `json:"file"`
	Range      rang          `json:"range"`
	Label      string        `json:"label,omitempty"`
	Hints      []hint        `json:"hints,omitempty"`
	Details    []detail      `json:"details,omitempty"`
	Highlights []highlight   `json:"highlights,omitempty"`
	Info       klarerrs.Info `json:"info,omitempty"`
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
	File    string `json:"file"`
	Range   rang   `json:"range"`
	Message string `json:"message"`
}
type highlight struct {
	Range   rang   `json:"range"`
	Message string `json:"message"`
}
type code struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

func convertCode(cd klarerrs.Code) code {
	return code{Name: cd.Format(), ID: int(cd)}
}

func convertHints(hints []klarerrs.Hint) []hint {
	hs := make([]hint, len(hints))
	for i, hn := range hints {
		hs[i] = hint{hn.Message}
	}
	return hs
}

func convertDetails(details []klarerrs.Detail) []detail {
	ds := make([]detail, len(details))
	for i, dt := range details {
		ds[i] = detail{
			File:    dt.File,
			Range:   rang{pos(dt.Range.Start), pos(dt.Range.End)},
			Message: dt.Message,
		}
	}
	return ds
}

func convertHighlights(highlights []klarerrs.Highlight) []highlight {
	hs := make([]highlight, len(highlights))
	for i, ht := range highlights {
		hs[i] = highlight{
			Range:   rang{pos(ht.Range.Start), pos(ht.Range.End)},
			Message: ht.Message,
		}
	}
	return hs
}
