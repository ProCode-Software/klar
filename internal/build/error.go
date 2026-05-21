package build

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
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
	default:
		panic(fmt.Sprintf("no InterfaceError message for %d", err.Code))
	}
}

func PrintInterfaceErr(err *InterfaceError) {
	if IsKlonError(err) {
		PrintKlonError(err)
		return
	}
	main, detail := err.PrettyError()
	cli.Error(ansi.Sprintf("<**>%s</**>%s", main, detail))
}

func PrintKlonError(err *InterfaceError) {
	dir, name := filepath.Split(err.Value)
	cli.Error(ansi.Sprintf("<**>Failed to parse <dim>%s</dim><c>%s</c>:</**>", dir, name))
	// TODO: use reporter
	klonErr := err.Err.(*klon.Error)
	fmt.Printf("%s "+ansi.Dim("%s")+"\n", klonErr.Text, klonErr.Range.Start)
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
