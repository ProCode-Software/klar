package build

import (
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/target"
)

// TODO: add documentation which will also be added to schema
// TODO: transform features in klarml unmarshaller, such as mapping strings
// to enums

type BuildFile struct {
	// The environment to build for.
	Target target.Target
	// Enable verbose logging during build. Useful for bug reporting.
	Verbose bool
	// Actions to run before building.
	PreBuild []any
	// Actions to run after build.
	PostBuild []any

	// Compiler warnings to hide during build.
	SuppressWarnings []string
	// Compiler warnings that should be shown as errors and fail the build.
	WarningsAsErrors []string

	// Additional build configurations to override the top-level. All configurations are run.
	Configurations []*FileConfiguration
	FileConfiguration
}

type FileConfiguration struct {
	// Klar source files and directories that should be compiled. Glob patterns are
	// allowed. If omitted, the entire package is compiled.
	Input []string
	// Output file or directory paths compiled files should be written inside. A file
	// pattern can be provided, otherwise the outputs must match the number of inputs.
	Output []string
	// Whether a full package structure with a package.json file should be built.
	EmitPackage bool
	// Rebuild files when they are modified. Useful for development.
	Watch bool
	// Whether comments in source files should be removed from built files. This value
	// does not influence JSDoc comment generation.
	StripComments bool
	// Mapping of source paths to output paths. Glob patterns may be used
	Paths map[string]string
	// Options when building JavaScript files
	JS *FileJSOptions
	// Options for the type checker
	Checker *FileCheckerOptions
	// Whether all files in output folders should be deleted before build. Disabled
	// if the output folder is the project root. Compiled files overwrite existing
	// files regardless.
	CleanOutputDir bool
	// Options for building assets
	Assets *FileAssetOptions
	// Output symbolic link to files that are being copied, such as assets and
	// `node_modules`. These save space by avoiding duplicating files, but output
	// files are not safe to modify. On Windows, junction links are used instead.
	UseSymlinks bool
}

type FileAssetOptions struct {
	// File extensions that should be copied to the output directory.
	Extensions []string
	// Directory that assets should be copied to. Relative to the build output directory.
	AssetDir string
	// Whether .klarml files should be transformed to .json files.
	KlarmlToJSON bool
}

type FileJSOptions struct {
	// Whether TypeScript declarations (.d.ts files) should be generated.
	// Recommended for all JavaScript libraries so users can get code
	// completion for your library in supporting IDEs.
	Declaration bool
	// Bundle all TypeScript declarations into one file.
	BundleDeclaration bool
	// Add JSDoc comments to exports in the resulting JavaScript files.
	JSDoc bool
	// Directory that *.d.ts files should be built to. If `bundleDeclaration` is on,
	// a single file will be generated here.
	DeclarationPath string
	// Enable experimental ECMAScript libraries. If enabled, generated JavaScript files
	// may also use experimental ECMAScript syntax.
	ESNext bool
	// TypeScript declaration libraries that should be loaded when type-checking. These
	// can be bundled with TypeScript, or paths to *.d.ts files.
	TypeScriptLibs []string
	// The module format to compile JavaScript files in. The file extension does not
	// change unless you modify your outputs. ESM is the standard JavaScript format, and
	// is the default and recommended option. CommonJS (`require()`) is not supported.
	Format js.ModuleFormat
	// How built JavaScript files should be bundled.
	Bundle js.BundleMode
	// Whether source map files should be created. If set to `true`, *.js.map files are
	// created alongside built JavaScript files. If set to `'inline'`, source maps are
	// stored as data URIs at the end of JavaScript files.
	Sourcemap any
	// The name of the global namespace for the compiled UMD module. Only supported if
	// 'format' is set to 'umd'.
	UMDNamespace string
	// Whether files should be printed by minimizing whitespace and line breaks,
	// reducing the size of JavaScript files.
	Minify bool
	// Code to add at the top of each built file, usually a comment.
	Banner string
	// Whether a `node_modules` directory should be created in the output directory.
	CopyNodeModules bool
	// Options for the dev server.
	Server *FileJSServerOptions
	// Global objects that should be made available to use in source files.
	Globals map[string]js.GlobalType
	// Path to a .d.ts file containing type definitions for items defined in `globals`.
	GlobalTypeDefs string
}

// In the dev server, links to compiled modules are made available.
type FileJSServerOptions struct {
	// Enable the dev server.
	Enabled bool
	// The HTML file to serve. This may also be a directory.
	Document string
	Port     int
	Host     string
}

type FileCheckerOptions struct {
	// Whether errors should be reported for 'when' statements that don't
	// cover all options. If set to 'enumsOnly', exhaustiveness is only
	// validated for 'when' statements that match enums.
	ValidateExhaustiveness ExhaustivenessOption
	// Whether all function declarations (not lambdas) should be required
	// to have explicit return types. TODO: This is always enabled for now.
	ExplicitReturnTypes bool
	// Whether the `Int` and `Float` should be treated as the same type.
	// Useful when compiling for JavaScript, where all numbers are floats.
	CoerceNumbers bool
	// Whether JavaScript externals should be checked that the export exists.
	// This is accomplished by importing the external JS file using the
	// project's default runtime and indexing the export name.
	ValidateExternals bool
}

// TODO: move to different package
type ExhaustivenessOption int

const (
	NoExhaustiveness ExhaustivenessOption = iota
	AllExhaustiveness
	EnumExhaustiveness
)
