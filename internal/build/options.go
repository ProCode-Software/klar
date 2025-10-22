package build

import (
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

type (
	InputKind int
	BuildMode int
	Flags     int16
)

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
)

const (
	CreateJSDoc Flags = 1 << iota
	CreateDeclaration
	Minify
	CreateSourceMap
	CopyNodeModules
	BundleDeclaration
	UseESNext
)

const (
	KindDir InputKind = 1 << iota
	KindFile
	KindPackage = KindDir | (1 << iota)
	KindModule  = KindDir | (1 << iota)
)

type Options struct {
	Inputs       []Input
	Target       target.Double `arg:"target"`
	Outputs      []string
	OutputDir    string `arg:"output"`
	JS           *JSOptions
	AssetOptions *AssetOptions
	Paths        map[string]string
	Watch        bool `arg:"watch"`
	EmitPackage  bool
	// ProjectDir   string
}

type JSOptions struct {
	Bundle         js.BundleMode
	Format         js.ModuleFormat
	Flags          Flags
	Banner         string
	DeclarationDir string
	TypeScriptLibs []string
}

type Input struct {
	Kind InputKind
	Path string
}

type AssetOptions struct {
	Extensions   []string // Glob path of file name/extensions
	AssetDir     string
	KlarmlToJSON bool
}

type Compiler struct {
	Mode    BuildMode
	Verbose bool
	Errors  []errors.CompileError
	Options []*Options
	Project *module.ProjectInfo
	*log.Logger
}

func (o JSOptions) HasFlag(flag Flags) bool {
	return (o.Flags & flag) != 0
}

// Logging
// ==========
var KLAR_LOG_FILE *os.File

func (c *Compiler) InitLogger() (hasLogFile bool) {
	logFile := os.Getenv("KLAR_LOG_FILE")
	var out io.Writer
	switch {
	case logFile != "":
		file, err := os.Create(logFile)
		if err != nil {
			cli.Failure("Unable to open KLAR_LOG_FILE '"+logFile+"': ", err)
		}
		out, KLAR_LOG_FILE, hasLogFile = file, file, true
	case c.Verbose:
		out = os.Stderr
	default:
		out = io.Discard
	}
	c.Logger = log.New(out, "[compiler] ", log.Ltime)
	return
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
