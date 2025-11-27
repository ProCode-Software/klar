package build

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
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
	ErrFileInPackage    // File directly in package root
	ErrFileInPkgDir     // File in 'pkg' directory
	ErrNoKlarFiles      // No Klar files to compile in input
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
		return "Expected a module name after " + "<c>'@'</c>", ""
	case ErrNotAKlarFile:
		ext := filepath.Ext(err.Value)[1:]
		base := err.Value[:len(err.Value)-len(ext)]
		return "<c>" + base + "<under>" + ext + "</under></c> isn't a Klar file", ""
	case ErrIsADirectory:
		return "", ""
	case ErrTooManyErrors:
		return "There are too many errors", ""
	case ErrMaxModuleDepth:
		return "Only 4 submodules are allowed: ",
			fmt.Sprintf("<c>%s</c> has %d", err.Value, MaxModuleDepth+1)
	case ErrFileInPackage:
		return "A file isn't allowed in the package or project root: ",
			"I found " + "<c>" + err.Value + "</c>"
	case ErrFileInPkgDir:
		return "A file isn't allowed in the " + "<c>pkg</c>" + " directory, ",
			"but I found " + "<c>" + err.Value + "</c>"
	case ErrNestedKlarFolder:
		dir, base := filepath.Split(err.Value)
		dir = strings.TrimSuffix(dir, "/")
		if base == module.PackageFolder {
			return "Can't nest the " + "<c>" + base + "</c> directory: ",
				"I found it nested in " + "<c>" + dir + "</c>"
		}
		return "The " + "<c>" + base + "</c>" +
				" directory is only allowed in the project root, ",
			"but I found it in " + "<c>" + dir + "</c>"
	case ErrNoKlarFiles:
		return "I didn't find any Klar files to compile in " + "<c>" + err.Value + "</c>", ""
	}
	return "", ""
}
