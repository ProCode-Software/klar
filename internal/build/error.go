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

const (
	_                   = iota
	ErrModuleDescriptor // Invalid '@' in input path
	ErrNotAKlarFile     // Input file is not a klar file
	ErrIsADirectory     // Path is a directory
	ErrTooManyErrors
)

type InterfaceError struct {
	Code  int
	Value string
	Err   error
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
	}
	return nil, ""
}
