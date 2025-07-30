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
	// Actions to run before building
	PreBuild  []any
	// Actions to run after build
	PostBuild []any

	// Compiler warnings to hide during build
	SuppressWarnings []string
	// Compiler warnings that should be shown as errors and fail the build
	WarningsAsErrors []string

	Configurations []*FileConfiguration
	FileConfiguration
}

type FileConfiguration struct {
	Input       []string
	Output      []string
	EmitPackage bool
	Watch       bool
	Paths       map[string]string
	// Options when building JavaScript files
	JS          *JSOptions
	Assets      []*AssetOptions
}

type FileAssetOptions struct {
	Extensions   []string
	AssetDir     string
	KlarmlToJSON string
}

type FileJSOptions struct {
	// Whether TypeScript declarations (.d.ts files) should be generated.
	// Recommended for all JavaScript libraries so users can get code
	// completion for your library in supporting IDEs.
	Declaration       bool
	// Bundle all TypeScript declarations into one file
	BundleDeclaration bool
	// Add JSDoc comments to exports in the resulting JavaScript files
	JSDoc             bool
	DeclarationPath    string

	// Enable experimental ECMAScript libraries 
	ESNext         bool
	// TypeScript declaration libraries that should be loaded when type-checking
	TypeScriptLibs []string

	Format          string
	Bundle          string
	Sourcemap       bool
	Minify          bool
	Banner          string
	CopyNodeModules bool
}
