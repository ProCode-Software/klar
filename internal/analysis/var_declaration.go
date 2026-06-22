package analysis

import (
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
)

// Variable represents a variable type.
type Variable struct {
	Type
	Object  *Object
	VarKind VariableKind
}

func (v *Variable) Underlying() Type { return v.Type }
func (v *Variable) Kind() Kind       { return v.Type.Kind() }

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
	Value ConstValue // TODO
}

// o's type is set to the returned variable.
func NewVariable(o *Object, kind VariableKind, typ Type) *Variable {
	vr := &Variable{Object: o, VarKind: kind, Type: typ}
	o.typ = vr
	return vr
}

func (c *Constant) Kind() Kind                        { return c.Type.Kind() }
func (c *Constant) Underlying() Type                  { return c.Type }
func (c *Constant) String() string                    { return "" }
func (c *Constant) StringWithName(name string) string { return "name := " + c.String() }

func (c *Checker) checkVarDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
	vr := o.typ.(*Variable)
	e := NewExprWithHint(o.context, *decl.rhsType, 0)
	c.checkExpr(decl.rhs, e) // TODO: be aware of destructures
	// TODO: Go calls check.initVar, which checks if the expression is untyped nil, sets untyped values to typed types, and calls check.assignment.
	vr.Type = e.Type
}

func (c *Checker) checkConstDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
	cnst := o.typ.(*Constant)

	// Check expression
	e := NewExprWithHint(o.context, *decl.rhsType, ConstExpr)
	c.checkExpr(decl.rhs, e) // TODO: be aware of destructures
	cnst.Value = e.ConstValue()
	if cnst.Value == nil {
		cnst.Value = UnknownConst{} // Ensure this is never nil if checking fails
	}
	cnst.Type = e.Type
	c.checkAssignment(e) // TODO: Go type checker calls check.assignment
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
