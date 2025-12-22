// Package glaspack defines the layout of glas.pack files, the Klar package manifest.
package glaspack

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
    // TODO: use version.Specifier and support ranges
    Klar version.Specifier
    // Supported targets this package can be built for. All code in this
    // module must be implemented for all of these targets, but '@target'
    // directives can be used to exclude individual objects.
    Target []target.Target
    // Options when targetting JavaScript
    JS *JavaScriptOptions
    // Mark this package as deprecated. Users will be warned when
    // attempting to install a deprecated package, optionally
    // displaying a message and alternative package to use instead.
    Deprecated *DeprecationOptions
    
    Dependencies DependencyList
    DevelopmentDependencies DependencyList 
}

type JavaScriptOptions struct {
	// The runtime used for running compiled JavaScript files for commands
	// such as 'klar run' and 'klar test'. This can be set to 'browser'
	// to run on an HTML page, a relative command that can be found in
	// PATH, or an absolute path to an executable
	DefaultRuntime string 
}

type DeprecationOptions struct {
	// Names of alternative packages that the user should install instead.
	Alternative []string
	// Optional deprecated message that is displayed to users when
	// they try to install this deprecated package.
	Message string
}

type DependencyList map[string]DependencySpecifier

type DependencySpecifier interface { depSpec() }

type VersionSpecifier struct {
    version.Specifier
}

type LocalSpecifier struct {
    Path string // Path to a package
}

type WorkspaceSpecifier struct {}

type NPMSpecifier struct {
    Version version.Specifier
    As string // Name of root module
}

// TODO: other providers and http