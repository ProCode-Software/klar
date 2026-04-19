package klarbuild

import (
	"github.com/ProCode-Software/klar/internal/target"
)

// TODO: add documentation which will also be added to schema
// TODO: transform features in klon unmarshaller, such as mapping strings
// to enums

type File struct {
	// Additional build configurations to override the top-level. All configurations are run.
	Configurations []*Configuration
	Configuration

	// Actions to run before building.
	PreBuild []any
	// Actions to run after build.
	PostBuild []any

	// Compiler warnings to hide during build.
	SuppressWarnings []string
	// Compiler warnings that should be shown as errors and fail the build.
	WarningsAsErrors []string

	// The environment to build for.
	Target target.Target
	// Enable verbose logging during build. Useful for bug reporting.
	Verbose bool
}

type Configuration struct {
	// Mapping of source paths to output paths. Glob patterns may be used
	Paths map[string]string
	// Options for building assets
	Assets *AssetOptions
	// Options when building JavaScript files
	JS *JSOptions
	// Options for the type checker
	Checker *CheckerOptions

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
	// Whether all files in output folders should be deleted before build. Disabled
	// if the output folder is the project root. Compiled files overwrite existing
	// files regardless.
	CleanOutputDir bool
	// Output symbolic link to files that are being copied, such as assets and
	// `node_modules`. These save space by avoiding duplicating files, but output
	// files are not safe to modify. On Windows, junction links are used instead.
	UseSymlinks bool
}

type AssetOptions struct {
	// Directory that assets should be copied to. Relative to the build output directory.
	AssetDir string
	// File extensions that should be copied to the output directory.
	Extensions []string
	// Whether .klon files should be transformed to .json files.
	KlonToJSON bool
}

type JSOptions struct {
	// Global objects that should be made available to use in source files.
	Globals map[string]GlobalType
	// Options for the dev server.
	Server *JSServerOptions
	// Path to a .d.ts file containing type definitions for items defined in `globals`.
	GlobalTypeDefs string
	// Directory that *.d.ts files should be built to. If `bundleDeclaration` is on,
	// a single file will be generated here.
	DeclarationPath string
	// Code to add at the top of each built file, usually a comment.
	Banner string
	// The name of the global namespace for the compiled UMD module. Only supported if
	// 'format' is set to 'umd'.
	UMDNamespace string
	// TypeScript declaration libraries that should be loaded when type-checking. These
	// can be bundled with TypeScript, or paths to *.d.ts files.
	TypeScriptLibs []string
	// How built JavaScript files should be bundled.
	Bundle BundleMode
	// Whether source map files should be created. If set to `true`, *.js.map files are
	// created alongside built JavaScript files. If set to `'inline'`, source maps are
	// stored as data URIs at the end of JavaScript files.
	Sourcemap SourceMapMode
	// Add JSDoc comments to exports in the resulting JavaScript files.
	JSDoc bool
	// Enable experimental ECMAScript libraries. If enabled, generated JavaScript files
	// may also use experimental ECMAScript syntax.
	ESNext bool
	// Whether files should be printed by minimizing whitespace and line breaks,
	// reducing the size of JavaScript files.
	Minify bool
	// Whether a `node_modules` directory should be created in the output directory.
	CopyNodeModules bool
	// Bundle all TypeScript declarations into one file.
	BundleDeclaration bool
	// Whether TypeScript declarations (.d.ts files) should be generated for
	// all public exports. Recommended for all JavaScript libraries so users
	// can get code completion for your library in supporting IDEs.
	Declaration bool
}

// In the dev server, links to compiled modules are made available.
type JSServerOptions struct {
	// The HTML file to serve. This may also be a directory.
	Document string
	Host     string
	Port     int
	// Enable the dev server.
	Enabled bool
}

type CheckerOptions struct {
	// Whether errors should be reported for 'when' statements that don't
	// cover all options for a type. If set to 'enumsOnly', exhaustiveness
	// is only validated for 'when' statements that match enums.
	// Exhaustiveness is always required in 'when' expressions.
	ValidateExhaustiveness ExhaustivenessOption
	// Whether assertions (using the `!` operator after an expression to crash
	// if the value is `nil` or an error) should be allowed. Avoiding assertions
	// prevents obscure crashes in programs, requiring programs to
	// explicitly check values and crashout.
	AllowAssertions CheckedAssertionOption
	// Whether all list index expressions should return `Result` instead of
	// crashing when out of bounds.
	CheckedListIndexing bool
	// Whether the `Int` and `Float` should be treated as the same type.
	// Useful when compiling for JavaScript, where all numbers are floats.
	CoerceNumbers bool
	// Whether JavaScript externals should be checked that the export exists.
	// This is accomplished by importing the external JS file using the
	// project's default runtime and indexing the export name.
	ValidateExternals bool
	// Whether all `Result`s must be used or checked. If enabled, an error will
	// be reported if a `Result` value is unused or discarded, such as
	// via `_ = fn()` or calling `fn()` as a statement.
	CheckAllResults bool
}

// Determines the level of exhaustiveness checking for 'when' expressions.
type ExhaustivenessOption int

const (
	// Don't check for exhaustiveness, except in 'when' expressions (not statements).
	NoExhaustiveness ExhaustivenessOption = iota
	// Always check for exhaustiveness for all types.
	AllExhaustiveness
	// Require exhaustiveness for all types except 'Result'.
	AllExhaustivenessExceptResult
	// Require exhaustiveness only for enum types.
	EnumExhaustiveness
)

// Determines whether and when assertions are allowed in code.
type CheckedAssertionOption int

const (
	// Allow all assertions in code.
	AllowAssertions CheckedAssertionOption = iota
	// Prevent programs that use the assertion syntax from being compiled.
	// Programs must crashout explicitly. This prevents hidden crashes
	// in production code.
	DisallowAssertions
	// Require comments on all lines containing assertions
	// stating that you know what you're doing.
	AllowAssertionsWithComments
)

// The level of bundling to apply to source files when compiling to JavaScript.
type BundleMode int

const (
	// Preserve the file structure and don't bundle files.
	BundleOff BundleMode = iota
	// Bundle all source files into one file, and bundle the standard
	// library separately. Default behaviour
	BundleSource
	// Each module and std get their own files
	BundlePerModule
	// Bundle everything including the standard library into one file
	BundleStd
)

// The JavaScript type for global objects.
type GlobalType uint16

const (
	GlobalObject   GlobalType = 1 << iota // Object
	GlobalString                          // String
	GlobalNumber                          // Number
	GlobalFunction                        // Function
	GlobalArray                           // Array. Can be used with another type.
	GlobalBoolean                         // Boolean
	GlobalError                           // Error
	GlobalNull                            // null
	GlobalConst                           // Constant value. Can be used with another type.
)

type SourceMapMode int

const (
	// Don't generate source maps.
	SourceMapDisabled SourceMapMode = iota
	// Generate separate source map files for each built JavaScript file.
	SourceMapEnabled
	// Append source map data to the end of each file instead of
	// creating separate files.
	SourceMapInline
)
