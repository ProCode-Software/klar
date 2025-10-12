package build

import (
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/build/js"
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
	Target       target.Double
	Outputs      []string
	OutputDir    string
	JS           *JSOptions
	AssetOptions *AssetOptions
	Paths        map[string]string
	Watch        bool
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

func (c *Compiler) InitLogger(verbose any) {
	if verbose == true {
		c.Logger = log.New(os.Stderr, "[compiler] ", log.Ltime)
		return
	}
	c.Logger = log.New(io.Discard, "[compiler] ", log.Ltime)
}

// Equivalent to c.Logger.Println
func (c *Compiler) Log(v ...any) {
	c.Println(v...)
}
