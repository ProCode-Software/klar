package build

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
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
	main, det := err.PrettyError()
	return strings.Join(main, "") + det
}

func (err *InterfaceError) PrettyError() (main []string, det string) {
	switch err.Code {
	case ErrModuleDescriptor:
		return []string{"Expected a module name after ", ansi.Cyan("'@'")}, ""
	case ErrNotAKlarFile:
		ext := filepath.Ext(err.Value)[1:]
		base := err.Value[:len(err.Value)-len(ext)]
		return []string{ansi.Cyan(base + ansi.Underline(ext)), " isn't a Klar file"}, ""
	case ErrIsADirectory:
		return nil, ""
	case ErrTooManyErrors:
		return []string{"There are too many errors"}, ""
	case ErrMaxModuleDepth:
		return []string{"Only 4 submodules are allowed: "},
			fmt.Sprintf("%s has %d", ansi.Cyan(err.Value), MaxModuleDepth+1)
	case ErrFileInPackage:
		return []string{"A file isn't allowed in the package/project root: "},
			"I found" + ansi.Cyan(err.Value)
	case ErrFileInPkgDir:
		return []string{"A file isn't allowed in the ", ansi.Cyan("pkg"), " directory, "},
			"but I found " + ansi.Cyan(err.Value)
	case ErrNestedKlarFolder:
		dir, base := filepath.Split(err.Value)
		dir = strings.TrimSuffix(dir, "/")
		if base == "pkg" {
			return []string{"Can't nest the ", ansi.Cyan(base), " directory: "},
				"I found it nested in " + ansi.Cyan(dir)
		}
		return []string{
			"The ", ansi.Cyan(base),
			" directory is only allowed in the project root, ",
		}, "but I found it in " + ansi.Cyan(dir)
	}
	return nil, ""
}
