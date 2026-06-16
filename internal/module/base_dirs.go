package module

import "path/filepath"

// Directories in the root of a package or project, as defined by the [Klar Project Structure Spec].
//
// [Klar Project Structure Spec]: https://github.com/ProCode-Software/klar/tree/main/docs/ProjectStructure.md
var KlarPackageDirs = map[string]struct{}{
	".klar": {}, "src": {}, "cmd": {}, "shared": {}, "external": {}, "pkg": {},
	"recipes": {}, "scripts": {}, "generated": {}, "dist": {}, "docs": {},
}

// Directories in the root of the project. These are not allowed in subpackage directories.
var ProjectOnlyDirs = map[string]struct{}{
	".klar": {}, "pkg": {}, "shared": {},
}

const sep = string(filepath.Separator)

// Per Klar Project Structure Spec
const (
	ManifestFile = "glas.pack"
	BuildFile    = "klar.build"
	LockFile     = "glas.lock"

	PkgDir       = "pkg"
	LocalDataDir = ".klar"
	SrcDir       = "src"
	DistDir      = "dist"
	ExternalDir  = "external"
	SharedDir    = "shared"
	CmdDir       = "cmd"
	TestDir      = "test"
)

func IsPackageDir(name string) bool {
	_, ok := KlarPackageDirs[name]
	return ok
}