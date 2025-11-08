package build

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/module"
)

func ResolveInputs(inputs []string) ([]Input, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	res := make([]Input, 0, len(inputs))
	for _, input := range inputs {
		if len(input) == 0 {
			continue
		}
		var i Input
		switch {
		case input == "-": // stdin
			i = Input{Kind: KindStdin}
		case input[0] == '@':
			// Module name reference: @foo.bar
			if len(input) == 1 {
				return nil, errors.New("Expected a module name after '@'")
			}
			i = Input{Kind: KindModule, Name: input[1:]}
			// TODO: resolve klar.build
		default:
			info, err := os.Stat(input)
			if err != nil {
				return nil, &FilesystemError{"stat", input, err}
			}
			if !info.IsDir() { // File
				i = Input{Path: input, Kind: KindFile}
			} else {
				// Directory: module or package
				i = Input{Path: input, Name: filepath.Base(input)}
				if isPkg, err := module.IsPackage(input); err != nil {
					return nil, &FilesystemError{"get", "working directory", err}
				} else if isPkg {
					i.Kind = KindPackage
				} else {
					i.Kind = KindModule
				}
			}
			// Get path to closest klar.build file
			ResolveKlarBuild(&i)
		}
		res = append(res, i)
	}
	return res, nil
}

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

// Does nothing if not found.
func ResolveKlarBuild(i *Input) {
	const sep = string(filepath.Separator)
	if i.Kind&KindDir != 0 {
		// Look inside directory before outside
		klarBuild := i.Path + sep + module.BuildFileName
		if _, err := os.Stat(klarBuild); err == nil {
			i.KlarBuild = klarBuild
			return
		}
	}
	if i.Path != "" {
		dir := filepath.Dir(i.Path) // Remove file name or dir (already handled)
		for {
			klarBuild := dir + sep + module.BuildFileName
			if _, err := os.Stat(klarBuild); err == nil {
				i.KlarBuild = klarBuild
				return
			}
			newDir := filepath.Dir(dir)
			if _, ok := module.KlarProjectDirs[filepath.Base(dir)]; ok || newDir == dir {
				return
			}
			dir = newDir
		}
	}
}
