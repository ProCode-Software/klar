package git

import (
	"path/filepath"
	"strings"
)

func InstallDirFor(klarDataDir, repoURL string) string {
	return filepath.Join(klarDataDir, "packages", EncodeURL(repoURL))
}

func EncodeURL(s string) string {
	return strings.ReplaceAll(strings.TrimPrefix(s, "https://"), "/", "+")
}
