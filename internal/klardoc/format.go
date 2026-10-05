package klardoc

import "github.com/ProCode-Software/klar/internal/ast"

type ProjectKlarDocJSON struct {
	// From glas.pack
	// ====

	// The name of the root package. Can be empty
	Name string `json:"name,omitempty"`
	// The version of the project. Can be empty
	Version string `json:"version,omitempty"`
	// The description of the project. Can be empty
	Description string `json:"description,omitempty"`

	// The subpackages in the project, in the 'pkg' folder
	Packages []*Package `json:"packages,omitempty"`
	// The commands in the project, in the 'cmd' folder
	Commands []*Module `json:"modules,omitempty"`
	*Package           // Top-level package, in the 'src' folder
}

type Package struct {
	// The name of the root package.
	Name string `json:"name"`
	// The version of the project.
	Version string `json:"version"`
	// The description of the project.
	Description string `json:"description"`
	// All modules in the package. Includes modules in the 'src' and 'shared' folders.
	Modules []*Module `json:"modules,omitempty"`
}

type Module struct {
	// The import path of the module. The name of the module is the last part of this.
	ImportPath string `json:"importPath"`
	// Path to the module on the filesystem. Relative to the package root
	SourcePath string `json:"sourcePath"`
	// Module-level documentation comment (///)
	Doc string `json:"doc"`
	// The objects declared in the module
	Declarations []*Declaration `json:"declarations"`
	// Basenames of all Klar files in the module, including those without declarations
	Files []string `json:"files"`
}

type Declaration struct {
	Name string `json:"name"`
	// Line and column position of the declaration in the source file
	Location Location `json:"location"`
	DeclarationData
	Type DeclarationType `json:"type"`
	// Whether the object was exported
	Public bool `json:"public"`
	// Documentation comment for the declaration
	Doc string `json:"doc"`
}

type DeclarationType uint8

const (
	KindVariable DeclarationType = iota
	KindConstant
	KindFunction
	KindFunctionAlias
	KindStruct
	KindTypeAlias
	KindEnum
	KindInterface
	KindTag
)

type Location struct {
	FilePath string    `json:"filePath"`
	Position [2]uint32 `json:"position"`
}

type DeclarationData interface {
	decl()
}

type TagDeclaration struct {
	Implements *[]string `json:"implements"`
}

// Used for variables and constants
type VariableDeclaration struct {
	// Explicit or inferred type of the variable/constant
	Type ast.Type `json:"type"`
	// Formatted value of the variable/constant
	Value string `json:"value"`
}

type StructDeclaration struct{}

type StructField struct{}

func (*TagDeclaration) decl()      {}
func (*VariableDeclaration) decl() {}
