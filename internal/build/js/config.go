package js

type (
	BundleMode   int
	ModuleFormat int
)

const (
	BundleOff       BundleMode = iota
	BundleSource               // Default behaviour
	BundlePerModule            // Each module and std get their own files
	BundleStd                  // Bundle everything including stdlib into one file
)

// CommonJS is intentionally not supported by the Klar compiler
const (
	ModuleESM ModuleFormat = iota // Default and recommended
	ModuleUMD                     // For use in browsers
)
