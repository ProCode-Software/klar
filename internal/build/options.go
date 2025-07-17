package build

import (
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
)

type BuildMode int

type Flags int32

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
)

type Options struct {
	Target       target.Double
	ProjectDir   string
	OutputDir    string
	JS           *JSOptions
	AssetOptions *AssetOptions
	Paths        map[string]string
	Verbose      bool
	Watch        bool
	SingleFile   bool
}

type JSOptions struct {
	Bundle         js.BundleMode
	Format         js.ModuleFormat
	Flags          Flags
	Banner         string
	DeclarationDir string
}

type AssetOptions struct {
	Extensions []string // Glob path of file name/extensions
	AssetDir   string
}

type Compiler struct {
	Mode   BuildMode
	Errors []errors.KlarError
	*Options
}

func (o JSOptions) HasFlag(flag Flags) bool {
	return (o.Flags & flag) != 0
}
