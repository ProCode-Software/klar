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
}

type JavaScriptOptions struct {
	// The runtime used to 
	DefaultRuntime string 
}