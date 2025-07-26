package build

import (
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
)

type (
	InputKind int
	BuildMode int
	Flags     int32
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
	// ProjectDir   string
}

type JSOptions struct {
	Bundle         js.BundleMode
	Format         js.ModuleFormat
	Flags          Flags
	Banner         string
	DeclarationDir string
}

type Input struct {
	Kind InputKind
	Path string
}

type AssetOptions struct {
	Extensions []string // Glob path of file name/extensions
	AssetDir   string
}

type Compiler struct {
	Mode    BuildMode
	Verbose bool
	Errors  []errors.KlarError
	Options []*Options
}

func (o JSOptions) HasFlag(flag Flags) bool {
	return (o.Flags & flag) != 0
}
