package build

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
)

type (
	InputKind int
	BuildMode int
)

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
	ModeTest                     // Resolve test files
)

const (
	KindDir InputKind = 1 << iota
	KindFile
	KindPackage = KindDir | (1 << iota)
	KindModule  = KindDir | (1 << iota)
	KindStdin   = KindFile | (1 << iota)
)

type Options struct {
	Inputs []Input
	klarbuild.File
}

type Input struct {
	Kind      InputKind
	Path      string // Filesystem path
	Name      string // Module or package name
	KlarBuild string // Path to klar.build file
}

type File struct {
	Path   string
	Tokens []lexer.Token
	AST    *ast.Program
}

type Module struct {
	Submodules []string // Submodule paths
	Files      []string // Klar file paths. Empty string = stdin
	Assets     []string // Non-Klar file paths
	Manifests  []string // Package-level and project-level glas.pack files
	// TODO: typechecked ast
}

type InputOptions struct {
	Modules []*Module
	Project *module.ProjectInfo
}

type Compiler struct {
	Mode                BuildMode
	verbose             bool
	StartTime           time.Time
	Errors              []errors.CompileError
	Options             []*Options
	PreBuild, PostBuild []any // TODO
	Opener              Opener
	openFiles           []*os.File

	inputs    map[*Input]*InputOptions
	modules   map[string]*Module // Module paths to modules
	flatFiles map[string]*File   // File paths to parsed ASTs and tokens

	errorPrinter *printer.Printer

	// From all configurations. TODO: better type
	SuppressWarnings, WarningsAsErrors []string
	*log.Logger
}

// File opening
// ============

type OpenFile struct {
	Size      int64
	ShortPath string
	io.ReadCloser
}

// Opener opens files for reading. The size parameter is the size of
// the file in bytes. [io.NopCloser] can be used to wrap a [io.Reader]
// if nothing needs to be done when closing.
type Opener interface {
	Open(name string) (*OpenFile, error)
}

// StdOpener implements [Opener] and is the standard implementation that reads
// Klar files on the system.
type StdOpener struct{ cwd string }

// Open implements [Opener]. Open returns the [*os.File], the size of the file in
// bytes when calling [os.File.Stat], and any error that occurred while opening.
func (o StdOpener) Open(name string) (f *OpenFile, err error) {
	fr, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := fr.Stat()
	if err != nil {
		return nil, err
	}
	relPath := name
	if !filepath.IsAbs(name) {
		relPath, err = filepath.Rel(o.cwd, name)
		if err != nil || strings.HasPrefix(relPath, "..") {
			// fallback to absolute path
			relPath = name
		}
	}
	return &OpenFile{
		Size:       stat.Size(),
		ShortPath:  relPath,
		ReadCloser: fr,
	}, nil
}

// Logging
// ==========

// Equivalent to c.Logger.Println
func (c *Compiler) Log(v ...any) {
	if c.verbose {
		c.Println(v...)
	}
}

func (c *Compiler) Logf(s string, v ...any) {
	if c.verbose {
		c.Logger.Printf(s, v...)
	}
}

func (c *Compiler) LogErrorf(s string, v ...any) {
	if c.verbose {
		c.Logf(ansi.Red("[error] ")+s, v...)
	}
}

func (c *Compiler) LogError(v ...any) {
	if c.verbose {
		v = append([]any{ansi.Red("[error]")}, v...)
		c.Println(v...)
	}
}

func (c *Compiler) CloseAll() {
	for _, file := range c.openFiles {
		file.Close()
	}
}
