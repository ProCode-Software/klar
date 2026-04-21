package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type DeclarationInfo struct {
	file *Context
	node ast.Statement
}

func (c *Checker) declareTopLevelObject(obj *Object, info *DeclarationInfo) {
	c.declare(c.rootContext, obj)
	c.moduleDecls[obj] = info
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
			File: orig.FilePath(),
			Highlight: errors.Highlight{
				Range:   orig.Range(),
				Message: errors.Quote(typeName) + " was declared here",
			},
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
	var methMap objectMap
	for _, meth := range methods {
		if existing := methMap.Insert(meth.obj); existing != nil {
			// Already declared
			err := errors.Range(errors.ErrRedeclared, meth.decl.Range)
			err.Details = append(err.Details, errors.Detail{
				File: existing.FileName(),
				Highlight: errors.Highlight{
					Range:   existing.Range(),
					Message: errors.Quote(meth.decl.Identifier.Name) + " was already declared here",
				},
			})
			c.error(err)
			continue
		}
		// TODO: make sure method doesn't have same name as a field
		def.AddMethod(meth.obj)
	}
}

func (c *Checker) checkDeclaration(o *Object) {
	if _, ok := c.objPathIndex[o]; ok {
		switch typ := o.typ.(type) {
		case *Variable, *Constant:
			if !c.isValidCycle(o) || typ.(Underlyer).Underlying() == nil {
				o.typ = KindInvalid
			}
		case *TypeName:
			if !c.isValidCycle(o) {
				o.typ = KindInvalid
			}
		case *Function:
			c.isValidCycle(o) // TODO: is this needed?
		default:
			panic("unreachable")
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
	case *Function:
	default:
		panic("unreachable")
	}
}

func (c *Checker) isValidCycle(o *Object) bool {
	return true
}
