//go:build windows

package main

import (
	"os"
	"path/filepath"
)

// When running `klar upgrade`, the previous version of Klar is moved to
// `klar.exe.old` (similar for glas.exe). On Windows, we can't delete or modify
// the old executable, so it was renamed to `klar.exe.old`. This ensures the old
// version is removed after the upgrade. This isn't an issue on other platforms.
func init() {
	exec, err := os.Executable()
	if err != nil {
		return // Don't prevent the user from running Klar if this fails
	}
	// Delete old version of both Klar and Glas
	for _, base := range [...]string{"klar.exe", "glas.exe"} {
		oldVersion := filepath.Join(filepath.Dir(exec), base+".old")
		_ = os.Remove(oldVersion)
	}
}
