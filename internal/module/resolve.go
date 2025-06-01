// Package module provides utilities for working with Klar modules
// and module resolution.
package module

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
)

func CacheDir() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		cli.InternalError("Cannot find Klar cache directory")
	}
	return filepath.Join(cache, "klar", "modules")
}

func JoinPath(base, module string) string {
	sep := string(filepath.Separator)
	module = strings.Replace(module, ".", sep, -1)
	return filepath.Join(base, module)
}

func IsWildcard(module string) bool {
	return strings.HasSuffix(module, "*")
}
