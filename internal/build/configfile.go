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
	PreBuild []any
	// Actions to run after build
	PostBuild []any

	// Compiler warnings to hide during build
	SuppressWarnings []string
	// Compiler warnings that should be shown as errors and fail the build
	WarningsAsErrors []string

	// Cannot be defined with a root configuration
	Configurations []*FileConfiguration
	FileConfiguration
}

type FileConfiguration struct {
	Input       []string
	Output      []string
	// Whether a package.json file should be built
	EmitPackage bool
	// Rebuild when a file changes
	Watch       bool
	// Mapping of source paths to output paths. Glob patterns may be used
	Paths       map[string]string
	// Options when building JavaScript files
	JS     *JSOptions
	// Options for building assets
	Assets []*AssetOptions
}

type FileAssetOptions struct {
	// File extensions that should be copied to the output directory
	Extensions   []string	
	// Directory that assets should be copied to. Relative to the build output directory.
	AssetDir     string
	// Whether .klarml files should be transformed to .json files
	KlarmlToJSON string
}

type FileJSOptions struct {
	// Whether TypeScript declarations (.d.ts files) should be generated.
	// Recommended for all JavaScript libraries so users can get code
	// completion for your library in supporting IDEs.
	Declaration bool
	// Bundle all TypeScript declarations into one file
	BundleDeclaration bool
	// Add JSDoc comments to exports in the resulting JavaScript files
	JSDoc           bool
	// Directory or file (if BundleDeclaration is enabled) path that .d.ts files should be built to
	DeclarationPath string

	// Enable experimental ECMAScript libraries. Also applies to generated JavaScript files.
	ESNext bool
	// TypeScript declaration libraries that should be loaded when type-checking
	TypeScriptLibs []string

	Format          string
	Bundle          string
	Sourcemap       bool
	Minify          bool
	Banner          string
	CopyNodeModules bool
}

type FileJSServerOptions struct {
	Enabled bool
	Document string // the HTML file
	Port int
	Host string
}