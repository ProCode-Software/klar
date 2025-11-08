package build

import (
	"io/fs"
	"path/filepath"
)

func (c *Compiler) ResolveModules() (err error) {
	c.ModuleMap = make(map[*Input]*Module, len(c.Options))
	c.Modules = make(map[string]*Module, len(c.Options))

	for _, opt := range c.Options {
		for i := range opt.Inputs {
			inp := &opt.Inputs[i]
			switch inp.Kind {
			case KindPackage:
				continue // TODO: skip for now
			case KindFile:
				c.ModuleMap[inp] = &Module{Files: []string{inp.Path}}
			case KindStdin:
				// Empty paths are stdin
				c.ModuleMap[inp] = &Module{Files: []string{""}}
			case KindModule:
				if err := c.moduleFromDir(inp.Path); err != nil {
					return err
				}
				// Link the input to its root module
				c.ModuleMap[inp] = c.Modules[inp.Path]
				continue
			}
			c.Modules[inp.Path] = c.ModuleMap[inp]
		}
	}

	return nil
}

func (c *Compiler) moduleFromDir(dir string) (err error) {
	m := &Module{
		Submodules: []string{},
		Files:      []string{},
	}
	c.Modules[dir] = m

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case path == dir:
			// Skip the initial directory
			return nil
		case d.IsDir():
			// Create a new module for this subdirectory
			m.Submodules = append(m.Submodules, path)
			return c.moduleFromDir(path)
		default:
			// Add the file to the root module
			m.Files = append(m.Files, path)
		}
		return nil
	})
}
