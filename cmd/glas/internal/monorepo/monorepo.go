package monorepo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
)

func ParseForFlag(projDir string, patterns []string) []string {
	// Ensure '**' isn't used in any pattern
	var negatePatterns []string
	for i, pat := range patterns {
		if strings.Contains(pat, "**") {
			cli.Failure("'**' can't be used in a '--for' pattern")
		}
		trimmed := strings.TrimLeft(pat, "!")
		// Add '!...' patterns to a separate list
		if len(trimmed) < len(pat) && (len(pat)-len(trimmed))%2 == 1 {
			negatePatterns = append(negatePatterns, trimmed)
			patterns = util.FastDelete(patterns, i)
		}
	}

	pkgDir := filepath.Join(projDir, module.PkgDir)
	if _, err := os.Stat(pkgDir); err != nil && errors.Is(err, fs.ErrNotExist) {
		// This project has no pkg/ dir
		cli.Failure(
			"Can't use '--for' flag outside of a monorepo.",
			"This project has only 1 package.",
		)
	}
	dirItems, err := os.ReadDir(pkgDir)
	if err != nil {
		cli.Failuref("Failed to list %s/pkg: %v", projDir, err)
	}

	// Find matches
	matches := make([]string, 0, len(dirItems))
	addMatches := func(patterns []string, negate bool) {
		if len(patterns) == 0 {
			return
		}
		for _, entry := range dirItems {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			for _, pat := range patterns {
				match, err := filepath.Match(pat, name)
				if err != nil && err == filepath.ErrBadPattern {
					cli.Failuref("Invalid pattern %q: %v", pat, err)
				}
				if match == !negate {
					matches = append(matches, name)
				}
			}
		}
	}
	addMatches(patterns, false)
	// Check against negated patterns. Packages that don't match WILL be included
	addMatches(negatePatterns, true)

	// Ensure at least 1 subpackage was matched
	if len(matches) == 0 {
		pkgNames := make([]string, 0, len(dirItems))
		for _, entry := range dirItems {
			if entry.IsDir() {
				pkgNames = append(pkgNames, entry.Name())
			}
		}
		cli.ColorErrorfln(
			"No packages matched the given '--for' patterns\n\n"+
				"The provided patterns were: <c!>%s</c!>\nThe available subpackages are: <y!>%s</y!>",
			strings.Join(patterns, ansi.ListSeparator(ansi.CodeBrightCyan, ",")),
			strings.Join(pkgNames, ansi.ListSeparator(ansi.CodeBrightYellow, ",")),
		)
	}
	return matches
}
