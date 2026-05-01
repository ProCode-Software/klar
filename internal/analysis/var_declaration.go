package analysis

import (
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
)

// Variable represents a variable type.
type Variable struct {
	*Object
	VarKind VariableKind
	Type    Type
}

func (v *Variable) Underlying() Type { return v.Type }

type VariableKind uint8

const (
	_              VariableKind = iota
	TopLevelVar                 // Module-level variable
	LocalVar                    // Locally declared variable
	SelfVar                     // self
	FuncParamVar                // Function parameter
	StructFieldVar              // Struct field
)

type Constant struct {
	Type  Type
	Value any // TODO
}

func (c *Constant) Kind() Kind                        { return c.Type.Kind() }
func (c *Constant) Underlying() Type                  { return c.Type }
func (c *Constant) String() string                    { return "" }
func (c *Constant) StringWithName(name string) string { return "name := " + c.String() }

func (c *Checker) checkVarDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}

func (c *Checker) checkConstDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}

// IsConst returns true if the given name is a constant name
// (all uppercase). Digits and underscores are allowed.
func IsConst(name string) bool {
	for _, r := range name {
		// Some characters, like CJK, are neither upper nor lower case. Allow them.
		if unicode.IsLower(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
