package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type DeclarationInfo struct {
	Attributes *Attributes
	file       *Context
	node       ast.Statement
	// For variable/const declaration. Not destructured
	rhs ast.Expression
	// For variable/const declaration. Set when rhs is checked.
	rhsType *Type
}

func (c *Checker) declareTopLevelObject(obj *Object,
	attrs *[]ast.Statement, info *DeclarationInfo,
) {
	c.declare(c.rootContext, obj)
	c.moduleDecls[obj] = info
	if attrs != nil {
		info.Attributes = c.parseAttributes(*attrs)
		*attrs = nil
	}
	obj.order = uint32(len(c.moduleDecls))
}

func (c *Checker) declare(ctx *Context, obj *Object, flags ...Flag) {
	if obj.name == "_" {
		return
	}
	if existing := ctx.Declare(obj, flags...); existing != nil {
		// Declared already
		err := errors.Range(errors.ErrRedeclared, obj.rang)
		err.Params = errors.ErrorParams{
			"existing":       existing.FileRange(),
			"name":           obj.name,
			"existingIsType": existing.IsTypeDecl() && !obj.IsTypeDecl(),
		}
		c.fileError(err, obj.file)
	}
}

// TODO
func (c *Checker) collectMethods(ctx *Context, typeName string, methods []methodInfo) {
	if len(methods) == 0 {
		return
	}
	// TODO: check that the method is in the same scope as the declaration
	self := ctx.Lookup(typeName)
	if self == nil {
		// typeName was declared in a different scope from the method
		orig := ctx.LookupRecursive(typeName)
		det := errors.Detail{
			File:    orig.FilePath(),
			Range:   orig.Range(),
			Message: errors.Quote(typeName) + " was declared here",
		}
		for _, meth := range methods {
			err := errors.Node(errors.ErrMethodInOtherScope, meth.decl)
			err.Details = append(err.Details, det)
			c.error(err)
		}
		return
	}
	def := getDefined(self.typ)
	if def == nil {
		println("Method resolution: DEFINED TYPE is nil")
		return
	}
	// TODO: wrap Overloads in Functions
	var methMap mapObject
	addMethod := func(meth methodInfo) {
		if existing := methMap.Insert(meth.obj); existing != nil {
			// Already declared
			err := errors.Range(errors.ErrRedeclared, meth.decl.Range)
			err.Details = append(err.Details, errors.Detail{
				File:    existing.FileName(),
				Range:   existing.Range(),
				Message: errors.Quote(meth.decl.Identifier.Name) + " was already declared here",
			})
			err.SetParam("kind", "method") // TODO
			c.error(err)
			return
		}
		// TODO: make sure method doesn't have same name as a field
		def.AddMethod(meth.obj)
	}
	// Non-aliases are declared before aliases
	var aliases []methodInfo
	for _, meth := range methods {
		if meth.alias != nil {
			aliases = append(aliases, meth)
		} else {
			addMethod(meth)
		}
	}
	// Now, method aliases
	for _, meth := range methods {
		addMethod(meth)
	}
}

func (c *Checker) checkDeclaration(o *Object) {
	if _, ok := c.objPathIndex[o]; ok {
		switch typ := o.typ.(type) {
		case *Variable, *Constant:
			if !c.isValidCycle(o) || typ.(Underlyer).Underlying() == nil {
				o.typ = InvalidType
			}
		case *TypeName:
			if !c.isValidCycle(o) {
				o.typ = InvalidType
			}
		case *Function:
			c.isValidCycle(o) // TODO: is this needed?
		default:
			panic(fmt.Sprintf("unhandled declaration type: %T", o.typ))
		}
	}

	if ut, ok := o.typ.(Underlyer); ok && ut.Underlying() != nil {
		return // Blue, already checked
	}

	// White, not checked yet
	c.pushToPath(o)
	defer c.popPath()

	decl := c.moduleDecls[o]
	switch o.typ.(type) {
	case *Variable:
		c.checkVarDecl(o, decl)
	case *Constant:
		c.checkConstDecl(o, decl)
	case *TypeName:
		c.checkTypeDecl(o, decl)
		// TODO: collect methods
	case *Function:
		c.checkFuncDecl(o)
	case *Overload:
		return // Overloads are part of functions
	default:
		panic(fmt.Sprintf("unhandled declaration type: %T", o.typ))
	}
}

func (c *Checker) isValidCycle(o *Object) bool {
	return true
}
