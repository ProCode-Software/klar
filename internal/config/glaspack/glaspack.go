// Package glaspack defines the layout of glas.pack files, the Klar package manifest.
package glaspack

import (
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type Manifest struct {
	// The name of the package. The name can be any valid Klar identifier,
	// with an optional scope at the beginning, separated with a slash `/`.
	Name string
	// A short description of what your package does and includes.
	Description string
	// The version of the package, starting with 'v'. It must follow Klar's
	// semantic versioning.
	Version version.Version
	// The minimum version of the Klar compiler this package can
	// be built with. Users with a lower version than this can't build
	// (and thus install) this package. Features introduced in newer Klar
	// versions can't be used in this package's code.
	//
	// `*` or `latest` aren't allowed.
	Klar version.Specifier
	// Supported targets this package can be built for. All code in this
	// module must be implemented for all of these targets, but '@target'
	// directives can be used to exclude individual objects.
	Target []target.Target
	// Additional links to display in this package's documentation.
	Links []*Link
	// Paths to exclude when this package is installed. Glob patterns are supported.
	Exclude []string
	// Options when targeting JavaScript ('js', 'node', 'deno', 'bun')
	JS *JavaScriptOptions
	// Permissions passed to the Deno runtime when targeting Deno or
	// when it is set as the default runtime.
	//
	// See: https://docs.deno.com/runtime/fundamentals/security/#permissions
	DenoPermissions *DenoPermissions
	// Mark this package as deprecated. Users will be warned when
	// attempting to install a deprecated package, optionally
	// displaying a message and alternative package to use instead.
	Deprecated *DeprecationOptions
	// Packages that are needed to build this package and are installed
	// alongside this package.
	Dependencies DependencyList
	// Packages that are only needed as build tools and aren't included
	// when this package is installed normally.
	DevelopmentDependencies DependencyList // TODO: devDependencies instead?
	// Set namespaces (values) for installed packages (keys). When a package's
	// namespace is set, all of its modules will be qualified under the
	// namespace. For example, by setting package `pkgA`'s namespace to `pkga`:
	// 	pkgA: pkga // Package -> Namespace
	// If the package provides modules `a` and `b`, they will be imported as
	// `pkga.a` and `pkga.b`.
	// This is useful for changing the Klar-generated names of NPM packages,
	// or resolving conflicts with import paths.
	ImportPaths map[string]string // Keys: package names, Values: import bases
}

type Link struct{ Label, URL string }

type JavaScriptOptions struct {
	// The runtime used for running compiled JavaScript files for commands
	// such as 'klar run' and 'klar test'. This can be set to 'browser' to
	// run on an HTML page, a relative command that can be found in PATH, or
	// an absolute path to an executable. Only supported on the general 'js' target.
	DefaultRuntime string
	// Command line arguments and flags passed to the default JavaScript
	// runtime when running 'klar run' and 'klar test' respectively.
	RunFlags, TestFlags []string
	// Whether the same arguments are set for both 'klar run' and
	// 'klar build'. If enabled, only one of those options may be set.
	SameRunAndTestFlags bool
}

type DenoPermissions struct {
	All                                           bool
	Read, Write, Env, Net, Exec, Sys, FFI, Import *DenoAllowList
}

type DenoAllowList struct {
	All         bool     // when set to boolean
	Allow, Deny []string // when set to object
}

type DeprecationOptions struct {
	// Names of alternative packages that the user should install instead.
	Alternative []string
	// Optional deprecated message that is displayed to users when
	// they try to install this deprecated package.
	Message string
}

// Dependencies
// =====

type (
	DependencyList      []DependencyCoder
	DependencySpecifier interface{ depSpec() }
)

// LocalSpecifier specifies that a dependency is a local package.
type LocalSpecifier struct {
	Path string // Path to a package or subpackage
}

// WorkspaceSpecifier specifies that a dependency is found as
// a subpackage in the current project.
type WorkspaceSpecifier struct{ Subpackage string }

// NPMSpecifier specifies that a dependency is from NPM.
type NPMSpecifier struct {
	Name    string
	Version version.Specifier
}

// GitSpecifier specifies that a package is on a Git repo. This
// is the default specifier
type GitSpecifier struct {
	URL        string
	Subpackage string // Specified via `@pkg`
	RefKind    GitRefKind
	Ref        string             // Commit (`+hash`) or branch (`branch`)
	Version    *version.Specifier // If [RefKind] is [RefKindTag]
}

type GitRefKind int

const (
	BranchRef GitRefKind = iota // @branch. Uses the latest commit on the branch.
	TagRef                      // @tag
	CommitRef                   // @commit
)

// Implements [DependencySpecifier]
func (*GitSpecifier) depSpec()       {}
func (*LocalSpecifier) depSpec()     {}
func (*WorkspaceSpecifier) depSpec() {}
func (*NPMSpecifier) depSpec()       {}
