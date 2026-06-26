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
	"github.com/ProCode-Software/klar/internal/util/graph"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
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

	ErrModuleDescriptor     // Invalid '@' in input path
	ErrNotAKlarFile         // Input file is not a klar file
	ErrTestInput            // Test input provided in non-test mode
	ErrIsADirectory         // Path is a directory
	ErrTooManyErrors        // More than 10 errors globally
	ErrMaxModuleDepth       // No more than 4 submodules
	ErrNestedKlarFolder     // Klar project directory nested in a pkg folder
	ErrFileInRoot           // File directly in package root
	ErrFileInPkgDir         // File in 'pkg' directory
	ErrNoKlarFiles          // No Klar files to compile in input
	ErrNothingToCompile     // No inputs
	ErrMisplacedTest        // Test file in a non-test directory
	ErrLexer                // Lexer error
	ErrInvalidConfig        // Failed to parse configuration
	ErrKlarVersion          // Compiler version too old to compile a package
	ErrDepCycle             // Dependency cycle
	ErrDepNotFound          // Dependency not found or installed
	ErrInternalCompileError // Internal modules failed to compile
	ErrNoManifest           // No manifest found
)

type InterfaceError struct {
	Code    InterfaceErrorCode
	Value   string
	Err     error
	Detail  any
	noColor bool // Don't use color for the detail
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
			fmt.Sprintf("<c>%s</c> has %d", err.Value, module.MaxModuleDepth+1)
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
	case ErrTestInput:
		return fmt.Sprintf(
			"Can't pass <c>%s</c> as an input outside of", err.Value,
		), "<m>klar test</m>"
	case ErrDepCycle:
		cycleErr := err.Err.(*graph.CycleError[string])
		if len(cycleErr.Cycle) == 1 {
			// Self-cycle
			return fmt.Sprintf("<m>%s</m> imports itself", cycleErr.Cycle[0]), ""
		}
		return "Import cycle found: ", fmt.Sprintf(
			"<m>%s → %s</m>",
			strings.Join(cycleErr.Cycle, " → "), cycleErr.Cycle[0],
		)
	case ErrDepNotFound:
		pkg := err.Value
		var msg string
		if err.Detail == "npm" {
			msg = "Can't find the location where NPM dependency <c>" + pkg + "</c> is installed. "
		} else {
			msg = "Dependency <c>" + pkg + "</c> isn't installed. "
		}
		return msg, "Run <m>glas install</m> to install it."
	case ErrInternalCompileError:
		err.noColor = true
		var posInfo string
		if err, ok := err.Err.(*klarerrs.Error); ok {
			posInfo = fmt.Sprintf(" (%s:%s)", err.File, err.Range.Start)
		}
		return "An internal compile error occurred: ", fmt.Sprintf(
			"This isn't your fault; please report an issue on GitHub.\n"+
				"The error is:\n    %v%s",
			err.Err, posInfo,
		)
	case ErrNoManifest:
		return "Project not found: ", fmt.Sprintf(
			"Can't find a <y>%s</y> file for <c>%s</c>", module.ManifestFile, err.Value,
		)
	default:
		panic(fmt.Sprintf("no InterfaceError message for %d", err.Code))
	}
}

func PrintInterfaceError(err *InterfaceError) {
	main, detail := err.PrettyError()
	if err.noColor {
		cli.Error(strings.TrimSuffix(main, " "), detail)
		return
	}
	cli.Error(ansi.Sprintf("<**>%s</**>%s", main, detail))
}

// PrintInterfaceOrKlonError is [PrintInterfaceError], but uses b's [Reporter] to
// report Klon errors if needed.
func (c *Compiler) PrintInterfaceOrKlonError(err *InterfaceError) {
	if IsKlonError(err) {
		c.PrintKlonError(err)
	} else {
		PrintInterfaceError(err)
	}
}

func (c *Compiler) PrintKlonError(ierr *InterfaceError) {
	kind := "configuration"
	if filepath.Base(ierr.Value) == module.ManifestFile {
		kind = "manifest"
	}
	cli.Error(ansi.Sprintf("<**>Failed to parse %s:</**>\n", kind))

	if err := c.printKlonDiagnostic(ierr.Err.(*klon.Error), ierr.Value, ""); err != nil {
		PrintInterfaceError(ierr)
		return
	}
}

func (c *Compiler) printKlonDiagnostic(err *klon.Error, file, title string) error {
	// Load tokens for reporter
	if !c.Reporter.FileLoaded(file) {
		absPath, err := filepath.Abs(file)
		if err != nil {
			absPath = file
		}
		c.Reporter.LoadFile(
			file,
			util.RelPath(c.WorkDir, absPath),
			makeKlonTokens(file),
		)
	}
	_, err2 := c.Reporter.Report(&errorWithFile{err, file, title})
	return err2
}

func (c *Compiler) PrintKlonWarnings(warn []*klon.Error, file string) {
	if len(warn) == 0 {
		return
	}

	title := "Configuration warning"
	if filepath.Base(file) == module.ManifestFile {
		title = "Manifest warning"
	}
	if len(warn) > 10 {
		warn = warn[:10]
		cli.Custom(
			ansi.BoldBrightYellow(title), "",
			fmt.Sprintf(
				ansi.BrightYellow("There are %d warnings; showing only the first 10:"),
				len(warn),
			), "\n",
		)
	}
	for _, err := range warn {
		if err := c.printKlonDiagnostic(err, file, title); err != nil {
			cli.Custom(ansi.BoldBrightYellow(title), err.Error())
		}
	}
	fmt.Println()
}

// errorWithFile adds a custom file to a [reporter.Error].
type errorWithFile struct {
	reporter.Error
	file  string
	title string
}

func (e *errorWithFile) FilePath() string { return e.file }
func (e *errorWithFile) Title() string {
	if e.title != "" {
		return e.title
	}
	return e.Error.Title()
}

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

func (c *Compiler) FailWithError(err error) {
	switch err := err.(type) {
	case *InterfaceError:
		c.PrintInterfaceOrKlonError(err)
	case *FilesystemError:
		cli.FailureError(err)
	default:
		cli.FailureError(err)
	}
	cli.Exit(1)
}

// hasErrs is whether errs contains errors that fail compilation. If all
// errs are warnings, or errs is empty, hasErrs is false.
func (c *Compiler) sendErrors(errs []*klarerrs.Error) (hasErrs, isMax bool) {
	if len(errs) == 0 {
		return false, false
	}
	c.collectMu.Lock()
	defer c.collectMu.Unlock()
	for _, err := range errs {
		if err.IsWarning() {
			c.Warnings = append(c.Warnings, err)
			// TODO: Warnings as errors
			continue
		}
		c.Errors = append(c.Errors, err)
		hasErrs = true
		if len(c.Errors) > MaxErrors {
			c.Errors = c.Errors[:MaxErrors]
			return true, true
		}
	}
	return
}

// PrintError prints an error to the error printer.
func (c *Compiler) PrintError(err *klarerrs.Error) (int64, error) {
	return c.Reporter.Report(err)
}

func (c *Compiler) PrintAllErrors(errs []*klarerrs.Error) {
	for _, err := range errs {
		c.PrintError(err)
	}
}
