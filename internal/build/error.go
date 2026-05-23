package build

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
	"github.com/ProCode-Software/klar/pkg/klon"
)

type FilesystemError struct {
	op   string
	Path string
	base error
}

func (err *FilesystemError) Error() string {
	return fmt.Sprintf("Failed to %s %s: %v", err.op, err.Path, err.base)
}
func (err *FilesystemError) Unwrap() error { return err.base }
func (err *FilesystemError) IsNotExist() bool {
	return err.op == "stat" && errors.Is(err.base, fs.ErrNotExist)
}

// InterfaceErrorCode represents the error type of an [InterfaceError].
type InterfaceErrorCode int

const (
	_ InterfaceErrorCode = iota

	ErrModuleDescriptor // Invalid '@' in input path
	ErrNotAKlarFile     // Input file is not a klar file
	ErrIsADirectory     // Path is a directory
	ErrTooManyErrors    // More than 10 errors globally
	ErrMaxModuleDepth   // No more than 4 submodules
	ErrNestedKlarFolder // Klar project directory nested in a pkg folder
	ErrFileInRoot       // File directly in package root
	ErrFileInPkgDir     // File in 'pkg' directory
	ErrNoKlarFiles      // No Klar files to compile in input
	ErrNothingToCompile // No inputs
	ErrMisplacedTest    // Test file in a non-test directory
	ErrLexer            // Lexer error
	ErrInvalidConfig    // Failed to parse configuration
)

type InterfaceError struct {
	Code   InterfaceErrorCode
	Value  string
	Err    error
	Detail any
}

func (err *InterfaceError) Error() string {
	main, detail := err.PrettyError()
	return ansi.Decolorize(main + detail)
}

func (err *InterfaceError) PrettyError() (main, detail string) {
	// Return strings with ANSI tags, not colorized yet.
	switch err.Code {
	case ErrModuleDescriptor:
		return "Expected a module name after <c>'@'</c>", ""
	case ErrNotAKlarFile:
		ext := filepath.Ext(err.Value)[1:]
		base := err.Value[:len(err.Value)-len(ext)]
		return "<c>" + base + "<under>" + ext + "</under></c> isn't a Klar file", ""
	case ErrIsADirectory:
		return "", ""
	case ErrTooManyErrors:
		return "There are too many errors", ""
	case ErrMaxModuleDepth:
		return "Only up to 4 submodules are allowed: ",
			fmt.Sprintf("<c>%s</c> has %d", err.Value, MaxModuleDepth+1)
	case ErrFileInRoot:
		return "A Klar file isn't allowed in the package or project root: ",
			"I found <c>" + err.Value + "</c>"
	case ErrFileInPkgDir:
		return "A Klar file isn't allowed in the <c>pkg</c> directory, ",
			"but I found <c>" + err.Value + "</c>"
	case ErrNestedKlarFolder:
		dir, base := filepath.Split(err.Value)
		dir = strings.TrimSuffix(dir, "/")
		if base == module.PkgDir {
			return "Can't nest the <c>" + base + "</c> directory: ",
				"I found it nested in <c>" + dir + "</c>"
		}
		return "The <c>" + base + "</c> directory is only allowed in the project root, ",
			"but I found it in <c>" + dir + "</c>"
	case ErrNoKlarFiles:
		return "I didn't find any Klar files to compile in <c>" + err.Value + "</c>", ""
	case ErrNothingToCompile:
		return "There's nothing to compile", ""
	case ErrLexer:
		return "An error occurred during tokenization: ", err.Err.Error()
	case ErrMisplacedTest:
		return "Test files must be in the <c>test</c> directory", ""
	case ErrInvalidConfig:
		return fmt.Sprintf("Failed to parse <c>%s</c>: ", err.Value), err.Err.Error()
	default:
		panic(fmt.Sprintf("no InterfaceError message for %d", err.Code))
	}
}

func PrintInterfaceError(err *InterfaceError) {
	main, detail := err.PrettyError()
	cli.Error(ansi.Sprintf("<**>%s</**>%s", main, detail))
}

// PrintInterfaceError is [PrintInterfaceError], but uses b's [Reporter] to
// report Klon errors if needed.
func (c *Compiler) PrintInterfaceError(err *InterfaceError) {
	if IsKlonError(err) {
		c.PrintKlonError(err)
	} else {
		PrintInterfaceError(err)
	}
}

func (c *Compiler) PrintKlonError(ierr *InterfaceError) {
	_, name := filepath.Split(ierr.Value)
	kind := "configuration"
	switch name {
	case "glas.pack":
		kind = "manifest"
	}
	cli.Error(ansi.Sprintf("<**>Failed to parse %s:</**>\n", kind))

	// Load tokens for reporter
	if !c.Reporter.FileLoaded(ierr.Value) {
		absPath, err := filepath.Abs(ierr.Value)
		if err != nil {
			absPath = ierr.Value
		}
		c.Reporter.LoadFile(
			ierr.Value,
			cli.RelPath(c.WorkDir, absPath),
			makeKlonTokens(ierr.Value),
		)
	}
	err := ierr.Err.(*klon.Error)
	if _, err := c.Reporter.Report(&errorWithFile{err, ierr.Value}); err != nil {
		PrintInterfaceError(ierr)
		return
	}
}

// errorWithFile adds a custom file to a [reporter.Error].
type errorWithFile struct {
	reporter.Error
	file string
}

func (e *errorWithFile) FilePath() string { return e.file }

func makeKlonTokens(filePath string) []lexer.Token {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var endCol int
	lastNl := bytes.LastIndexByte(content, '\n')
	switch {
	case lastNl < 0:
		endCol = utf8.RuneCount(content)
	case lastNl < len(content)-1:
		endCol = utf8.RuneCount(content[lastNl+1:])
	default:
		endCol = 1
	}
	return []lexer.Token{{
		Position: lexer.Position{1, 1},
		Source:   string(content),
		Attributes: map[string]any{"end": lexer.Position{
			Line: uint32(bytes.Count(content, []byte{'\n'})) + 1,
			Col:  uint32(endCol),
		}},
	}}
}

func IsMaxErrors(err error) bool {
	ie, ok := err.(*InterfaceError)
	return ok && ie.Code == ErrTooManyErrors
}

// IsKlonError returns true if the error was caused by a failure
// to parse a Klon file such as a config.
func IsKlonError(err error) bool {
	ie, ok := err.(*InterfaceError)
	if !ok {
		return false
	}
	_, ok = ie.Err.(*klon.Error)
	return ok
}
