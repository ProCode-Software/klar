package build

import (
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/target"
)

type Options struct {
	Target       target.Double
	ProjectDir   string
	OutputDir    string
	JSOptions    *JSOptions
	AssetOptions *AssetOptions
	Paths        map[string]string
	Verbose      bool
	Watch        bool
}

type JSOptions struct {
	Bundle          js.BundleMode
	Format          js.ModuleFormat
	Declaration     bool
	JSDoc           bool
	Minify          bool
	CreateSourceMap bool
	CopyNodeModules bool
	Banner          string
	DeclarationDir  string
}

type AssetOptions struct {
	Extensions []string // Glob path of file name/extensions
	AssetDir   string
}

type Build struct {
	*Options
}

