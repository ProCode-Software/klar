package manifest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
	"github.com/ProCode-Software/klar/pkg/klon"
)

// TODO: Use this functionality in the build package

var errorReporter *reporter.Reporter

var manifestCache = make(map[string]*glaspack.Manifest)

func GetPackageInfo() *module.PackageInfo {
	cwd, err := os.Getwd()
	if err != nil {
		cli.Failure("Failed to resolve current package:", err)
	}
	pkgDir, projDir := module.PackageRoot(cwd)

	// Parse manifests
	// ======

	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	klonError := func(err error, path string) {
		if klonErr, ok := err.(*klon.Error); ok {
			PrintKlonError(path, klonErr)
			cli.Exit(1)
		}
		cli.FailureDetailf("Failed to parse <c>%s</c>: ", err.Error(), path)
	}
	var (
		pkgManFile      = filepath.Join(pkgDir, module.ManifestFile)
		projManFile     = filepath.Join(projDir, module.ManifestFile)
		warn            []*klon.Error
		pkgMan, projMan *glaspack.Manifest
	)
	// Package manifest
	if cached, ok := manifestCache[pkgManFile]; ok {
		pkgMan = cached
	} else if exists(pkgManFile) {
		if pkgMan, warn, err = glaspack.Parse(pkgManFile); err != nil {
			klonError(err, pkgManFile)
		}
		PrintKlonWarnings(warn, pkgManFile)
		manifestCache[pkgManFile] = pkgMan
	}

	// Project-level manifest
	cached, ok := manifestCache[projManFile]
	switch {
	case ok && pkgDir != projDir:
		pkgMan = cached
	case pkgDir == projDir, !exists(projManFile):
		// Subpackages don't need a manifest if the project has one
		if pkgMan == nil {
			// No project nor package manifest
			cli.ErrNoManifest(projDir)
		}
		return module.NewPackageInfo(pkgDir, projDir, pkgMan)
	default:
		projMan, warn, err = glaspack.Parse(projManFile)
		if err != nil {
			klonError(err, projManFile)
		}
		PrintKlonWarnings(warn, projManFile)
		manifestCache[projManFile] = projMan
	}
	if pkgMan, err = glaspack.Merge(pkgMan, projMan); err != nil {
		klonError(err, projManFile)
	}
	return module.NewPackageInfo(pkgDir, projDir, pkgMan)
}

func PrintKlonWarnings(warn []*klon.Error, file string) {
	if len(warn) == 0 {
		return
	}

	title := "Configuration warning"
	if filepath.Base(file) == module.ManifestFile {
		title = "Manifest warning"
	}
	if fullLen := len(warn); fullLen > 10 {
		warn = warn[:10]
		cli.Warn(ansi.BrightYellow(fmt.Sprintf(
			"There are %d warnings; we're showing only the first 10",
			fullLen,
		)))
	}
	for _, err := range warn {
		if err := printKlonDiagnostic(err, file, title); err != nil {
			cli.CustomError(ansi.CodeBoldBrightYellow, title, err.Error())
		}
	}
	fmt.Println()
}

func PrintKlonError(file string, err *klon.Error) {
	kind := "configuration"
	if filepath.Base(file) == module.ManifestFile {
		kind = "manifest"
	}
	cli.ColorErrorfln("<**>Failed to parse %s:</**>\n", kind)

	if err := printKlonDiagnostic(err, file, ""); err != nil {
		// Fallback
		cli.Error(err.Error())
		return
	}
}

func printKlonDiagnostic(err *klon.Error, file, title string) error {
	getShortPath := func() string {
		if absPath, err := filepath.Abs(file); err == nil {
			if cwd, err := os.Getwd(); err == nil {
				return util.RelPath(cwd, absPath)
			}
		}
		return file
	}
	if errorReporter == nil {
		errorReporter = &reporter.Reporter{
			MaxLines:     3,
			Output:       os.Stderr,
			ColorPalette: reporter.DefaultColorPalette(),
			CharacterSet: reporter.DefaultCharacterSet(),
			UseColor:     !ansi.DisableColor,
		}
	}

	// Load tokens for reporter
	if !errorReporter.FileLoaded(file) {
		errorReporter.LoadFile(file, getShortPath(), makeKlonTokens(file))
	}
	_, err2 := errorReporter.Report(&errorWithFile{err, file, title})
	return err2
}

// errorWithFile adds a custom file to a [reporter.Error].
type errorWithFile struct {
	reporter.Error
	file  string
	title string
}

func (e *errorWithFile) FilePath() string { return e.file }
func (e *errorWithFile) Title() string {
	if e.title != "" {
		return e.title
	}
	return e.Error.Title()
}

func makeKlonTokens(filePath string) []lexer.Token {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var endCol int
	lastNl := bytes.LastIndexByte(content, '\n')
	switch {
	case lastNl < 0:
		endCol = utf8.RuneCount(content)
	case lastNl < len(content)-1:
		endCol = utf8.RuneCount(content[lastNl+1:])
	default:
		endCol = 1
	}
	return []lexer.Token{{
		Position: lexer.Position{1, 1},
		Source:   string(content),
		Attributes: map[string]any{"end": lexer.Position{
			Line: uint32(bytes.Count(content, []byte{'\n'})) + 1,
			Col:  uint32(endCol),
		}},
	}}
}

func IsMonorepoRoot(pi *module.PackageInfo) bool {
	if pi.Dir != pi.ProjectDir {
		return false
	}
	_, err := os.Stat(filepath.Join(pi.ProjectDir, module.PkgDir))
	return err == nil
}
