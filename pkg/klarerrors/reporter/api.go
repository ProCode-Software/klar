// Package reporter formats [CompileError] with colored file context and
// highlights. This is based on [miette], an implementation written in Rust.
//
// [miette]: https://github.com/zkat/miette
package reporter

import (
	"bytes"
	"io"
	"os"

	"github.com/ProCode-Software/klar/internal/lexer"
)

type Reporter struct {
	MaxLines     int           // If set to 0, MaxLines is set to 3.
	Output       io.Writer     // If set to nil, [os.Stderr] is used.
	ColorPalette *ColorPalette // If set to nil, colored tokens are disabled.
	CharacterSet *CharacterSet // If set to nil, [DefaultCharacterSet] is used.
	// Alternative error titles to display instead of the type name. Keys are error prefixes.
	ErrorNames map[int]string
	files      map[string]file
	buf        *bytes.Buffer
}

type file struct {
	tokens      []lexer.Token
	rel         string // Name used when printing
	lastLineTok int    // Index of first token on line of last reported error
}

// NewReporter returns a [*Reporter] with recommended settings.
func NewReporter() *Reporter {
	return &Reporter{
		MaxLines:     3,
		Output:       os.Stderr,
		ColorPalette: DefaultColorPalette(),
		CharacterSet: DefaultCharacterSet(),
		files:        make(map[string]file),
	}
}

// NewReporter returns a [*Reporter] with recommended settings.
func NewReporterTheme(colors *ColorPalette, chars *CharacterSet) *Reporter {
	return &Reporter{
		MaxLines:     3,
		Output:       os.Stderr,
		ColorPalette: colors,
		CharacterSet: chars,
		files:        make(map[string]file),
	}
}

// LoadFile loads the file and tokens into r. path is the file path as provided
// in the errors. rel is the path to be displayed when printed. If rel is an empty
// string, path is displayed. Errors with [CompileError.GetFile] path display
// tokens.
func (r *Reporter) LoadFile(path, rel string, tokens []lexer.Token) {
	if r.files == nil {
		r.files = make(map[string]file)
	}
	r.files[path] = file{tokens: tokens, rel: rel}
}

// RemoveFile unloads path with path's tokens from r.
// If a file with path is not found, RemoveFile is a no-op.
func (r *Reporter) RemoveFile(path string) {
	delete(r.files, path)
}
