package build

import (
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
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
	BuildFile
	// ProjectDir   string
}

type Input struct {
	Kind      InputKind
	Path      string // Filesystem path
	Name      string // Module or package name
	KlarBuild string // Path to klar.build file
}

type File struct {
	Path    string
	Tokens  []lexer.Token
	AST     *ast.Program
}

type Module struct {
	Submodules []string // Submodule paths
	Files []string // File paths
	// TODO: typechecked ast
}

type Compiler struct {
	Mode                BuildMode
	Verbose             bool
	Errors              []errors.CompileError
	Options             []*Options
	Project             *module.ProjectInfo
	PreBuild, PostBuild []any // TODO
	OpenFiles           []*os.File

	ModuleMap map[*Input]*Module
	Modules   map[string]*Module // Module paths to modules
	FlatFiles map[string]*File

	ErrorPrinter *printer.Printer

	SuppressWarnings, WarningsAsErrors []string // TODO: better type
	*log.Logger
}

// Logging
// ==========

// InitLogger sets c.Logger. If the $KLAR_LOG_FILE envionment variable is set,
// c.Logger is set to write to that file (regardless of the value of c.Verbose).
// If c.Verbose is false, c.Logger is set to [io.Discard]. Otherwise, c.Logger
// is set to [os.Stderr].
func (c *Compiler) InitLogger() {
	logFile := os.Getenv("KLAR_LOG_FILE")
	var out io.Writer
	switch {
	case logFile != "":
		file, err := os.Create(logFile)
		if err != nil {
			cli.Failure("Unable to open KLAR_LOG_FILE '"+logFile+"': ", err)
		}
		c.OpenFiles = append(c.OpenFiles, file)
		out = file
		c.Verbose = true
	case c.Verbose:
		out = os.Stderr
	default:
		out = io.Discard
	}
	c.Logger = log.New(out, "[compiler] ", log.Ltime)
}

// Equivalent to c.Logger.Println
func (c *Compiler) Log(v ...any) {
	if c.Verbose {
		c.Println(v...)
	}
}

func (c *Compiler) Errorf(s string, v ...any) {
	if c.Verbose {
		c.Printf("[error] "+s, v...)
	}
}

func (c *Compiler) CloseAll() {
	for _, file := range c.OpenFiles {
		file.Close()
	}
}
