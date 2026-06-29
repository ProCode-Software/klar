// Package reporter formats [*Error] with colored file context and
// highlights. This is based on the Rust crate [miette].
//
// [miette]: https://github.com/zkat/miette
package reporter

import (
	"bytes"
	"io"
	"os"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Error is an interface that represents an error to report.
type Error interface {
	Title() string     // The title of the error, such as "Error".
	Message() string   // The error message.
	ErrorCode() string // The error code, displayed after the message.
	IsWarning() bool   // Whether the error is a warning.

	FilePath() string       // The full path of the file where the error occurred.
	Location() ranges.Range // The start and end positions of the error in the file.
	MainHighlight() string  // The text to display after the main underline.

	// Additional file ranges to display after the error.
	ErrorDetails() []klarerrs.Detail
	// Additional underline locations in the file.
	ErrorHighlights() []klarerrs.Highlight
	// Hints to display after the error. A hint may display a diff.
	ErrorHints() []klarerrs.Hint
}

// A Reporter prints compile errors with colored file context and highlights.
type Reporter struct {
	MaxLines     int           // If set to 0, MaxLines is set to 3.
	Margin       int           // Number of spaces to add to the left of the box.
	Output       io.Writer     // If set to nil, [os.Stderr] is used.
	ColorPalette *ColorPalette // If set to nil, colored tokens are disabled.
	CharacterSet *CharacterSet // If set to nil, [DefaultCharacterSet] is used.
	UseColor     bool          // If true, color output is enabled.
	// If true, Reporter prints a separator before the next error.
	alreadyPrinted bool
	files          map[string]*file
	buf            *bytes.Buffer
}

type file struct {
	tokens      []lexer.Token
	shortPath   string // Name used when printing
	lastLine    uint32 // Line number of last reported error
	lastLineTok int    // Index of first token on line of last reported error
}

// NewReporter returns a [*Reporter] with recommended settings.
func NewReporter() *Reporter {
	return &Reporter{
		MaxLines:     3,
		Output:       os.Stderr,
		ColorPalette: DefaultColorPalette(),
		CharacterSet: DefaultCharacterSet(),
		files:        make(map[string]*file),
	}
}

// NewReporter returns a [*Reporter] with recommended settings.
func NewReporterTheme(colors *ColorPalette, chars *CharacterSet) *Reporter {
	return &Reporter{
		MaxLines:     3,
		Output:       os.Stderr,
		ColorPalette: colors,
		CharacterSet: chars,
		files:        make(map[string]*file),
	}
}

// LoadFile loads the file and tokens into r. path is the file path as provided in
// [Error.FilePath]. shortPath is the path to be displayed when printed. If shortPath
// is an empty string, path is displayed. If an error's [Error.FilePath]() == path,
// tokens are displayed. If the file is already loaded, rel and tokens are replaced.
func (r *Reporter) LoadFile(path, shortPath string, tokens []lexer.Token) {
	if r.files == nil {
		r.files = make(map[string]*file)
	}
	r.files[path] = &file{tokens: tokens, shortPath: shortPath}
}

// RemoveFile unloads path with path's tokens from r.
// If a file with path is not found, RemoveFile is a no-op.
func (r *Reporter) RemoveFile(path string) {
	delete(r.files, path)
}

// FileLoaded returns true if path is loaded in r.
func (r *Reporter) FileLoaded(path string) bool {
	return r.files != nil && r.files[path] != nil
}
