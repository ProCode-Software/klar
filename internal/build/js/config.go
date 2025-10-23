package js

type (
	BundleMode   int
	ModuleFormat int
	GlobalType   uint16
)

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

// CommonJS is intentionally not supported by the Klar compiler.
const (
	// Standard JavaScript format. Default and recommended
	ModuleESM ModuleFormat = iota
	// Keep all exports under a single global namespace. For use in browsers
	ModuleUMD
)

const (
	GlobalObject    GlobalType = 1 << iota // Object
	GlobalString                           // String
	GlobalNumber                           // Number
	GlobalFunction                         // Function
	GlobalArray                            // Array
	GlobalBoolean                          // Boolean
	GlobalError                            // Error
	GlobalUndefined                        // undefined

	GlobalConstant // Constant value
)
