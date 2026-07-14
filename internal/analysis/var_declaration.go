package analysis

import (
	"fmt"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
)

// Variable represents a variable type.
type Variable struct {
	Type
	Object  *Object
	VarKind VariableKind
}

// o's type is set to the returned variable.
func NewVariable(o *Object, kind VariableKind, typ Type) *Variable {
	vr := &Variable{Object: o, VarKind: kind, Type: typ}
	o.typ = vr
	return vr
}

func (v *Variable) Underlying() Type { return v.Type }
func (v *Variable) Kind() Kind       { return v.Type.Kind() }
func (v *Variable) String() string {
	return v.Type.String()
	/*
		var kind, name string
		switch v.VarKind {
		default:
			kind = "var"
		case SelfVar:
			kind = "self"
		case FuncParamVar:
			kind = "param"
		case StructFieldVar:
			kind = "field"
		}
		if v.Object != nil && v.VarKind != SelfVar {
			name = " " + v.Object.name
		}
		return fmt.Sprintf("%s%s: %s", kind, name, v.Type)
	*/
}
func (*Variable) objKind() {}

type VariableKind uint8

const (
	_              VariableKind = iota
	LocalVar                    // Locally declared variable
	TopLevelVar                 // Module-level variable
	SelfVar                     // self
	FuncParamVar                // Function parameter
	StructFieldVar              // Struct field
	PipelineVar                 // value
)

type Constant struct {
	Type  Type
	Value ConstValue // TODO
}

func (c *Constant) Kind() Kind       { return c.Type.Kind() }
func (c *Constant) Underlying() Type { return c.Type }
func (*Constant) objKind()           {}
func (c *Constant) String() string   { return fmt.Sprintf("%s (%v)", c.Type, c.Value) }

func (c *Checker) checkVarDecl(o *Object) {
	var (
		vr    = o.typ.(*Variable)
		vinfo = o.info.varInfo
		val   = vinfo.rhs
		e     *Expr
	)
	vr.Type = InvalidType
	// Use the cached expression or check the RHS
	if *vinfo.rhsExpr != nil {
		e = *vinfo.rhsExpr
	} else {
		e = c.checkExpr(val, NewExpr(o.LookupContext()).withHint(vinfo.expType))
		e.Type = c.toTyped(e.Type, vinfo.expType, val, e.Context.File)
		*vinfo.rhsExpr = e
	}
	// TODO: Go calls check.initVar, which checks if the expression is untyped
	// nil, sets untyped values to typed types, and calls check.assignment.

	// Destructure the RHS
	for dest, typ := range c.followDestructure(
		vinfo.lhs, e.Type, e.Context.File, val.GetRange(), true,
	) {
		// TODO: Evaluate followDestructure only once per vinfo.rhsExpr, and cache
		// the types of other variables using the same rhsExpr.
		if sym := dest.(*ast.Symbol); sym.Identifier == o.name {
			vr.Type = typ
			break
		}
	}
	if vr.Type == nil {
		panic(o.name + " not yielded by followDestructure or it yielded a nil Type")
		// vr.Type = InvalidType
	}
}

func (c *Checker) checkConstDecl(o *Object) {
	var (
		cnst  = o.typ.(*Constant)
		vinfo = o.info.varInfo
		val   = vinfo.rhs
		e     *Expr
	)
	// Use the cached expression or check the RHS
	if *vinfo.rhsExpr != nil {
		e = *vinfo.rhsExpr
	} else {
		e = c.checkExpr(val, NewExpr(o.LookupContext(), constExpr).withHint(vinfo.expType))
		e.Type = c.toTyped(e.Type, vinfo.expType, val, e.Context.File)
		*vinfo.rhsExpr = e
	}

	// TODO: destructure
	cnst.Value = e.ConstValue()
	if cnst.Value == nil {
		cnst.Value = UnknownConst{} // Ensure this is never nil if checking fails
	}
	cnst.Type = e.Type
	// TODO: Go type checker calls check.assignment
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
