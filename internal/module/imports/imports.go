package imports

import (
	"slices"
	"strings"
)

type ImportPath []string

func (p ImportPath) String() string {
	return strings.Join(p, ".")
}

func (p ImportPath) Namespace() string {
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1]
}

// Import base paths that are considered part of the standard library.
var StdlibImports = []string{"klar"}

// IsStdlib returns true if the import path is part of the standard library.
// This is true if the base import path is in [StdlibImports].
func (p ImportPath) IsStdlib() bool {
	if len(p) == 0 {
		return false
	}
	return slices.Contains(StdlibImports, p[0])
}

// IsPrivate returns true if the import path is private (contains parts
// that start with "_").
func (p ImportPath) IsPrivate() bool {
	return slices.ContainsFunc(p, func(d string) bool {
		return strings.HasPrefix(d, "_")
	})
}
