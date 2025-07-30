package build

import "github.com/ProCode-Software/klar/internal/target"

// TODO: add documentation which will also be added to schema
// TODO: transform features in klarml unmarshaller, such as mapping strings
// to enums

type BuildFile struct {
	// The environment to build for
	Target target.Target
	// Enable verbose logging during build. Useful for bug reporting.
	Verbose bool
	// Actions to run before
	PreBuild  []any
	PostBuild []any

	Configurations []*FileConfiguration
	FileConfiguration
}

type FileConfiguration struct {
	Input       []string
	Output      []string
	EmitPackage bool
	Watch       bool
	Paths       map[string]string
	JS          *JSOptions
	Assets      []*AssetOptions
}

type FileAssetOptions struct {
	Extensions   []string
	AssetDir     string
	KlarmlToJSON string
}

type FileJSOptions struct {
	Declaration       bool
	BundleDeclaration bool
	JSDoc             bool
	DeclarationDir    string

	ESNext         bool
	TypeScriptLibs []string

	Format          string
	Bundle          string
	Sourcemap       bool
	Minify          bool
	Banner          string
	CopyNodeModules bool
}
