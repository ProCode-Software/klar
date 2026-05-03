package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
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
	attrs *[]*ast.Attribute, info *DeclarationInfo,
) {
	c.declare(c.rootContext, obj)
	c.moduleDecls[obj] = info
	if attrs != nil {
		info.Attributes = c.parseAttributes(*attrs, attrTargetKindOf(info.node))
		*attrs = nil
	}
	obj.order = uint32(len(c.moduleDecls))
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
		return // Blue, already checked (TODO: is this correct?)
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
		c.checkCustomTypeDecl(o, decl)
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

// collectMethods associates methods with the type with name typeName. The
// receiver for the methods is looked up within the context and validated.
func (c *Checker) collectMethods(ctx *Context, typeName string, methods []methodInfo) {
	if len(methods) == 0 {
		return
	}
	selfObj := ctx.Lookup(typeName)
	if selfObj == nil {
		c.validateReceiver(typeName, ctx.LookupRecursive(typeName), methods, true)
		return
	}
	if !c.validateReceiver(typeName, selfObj, methods, false) {
		return
	}
	self := Underlying(selfObj.typ).(MethodAdder)
	for _, meth := range methods {
		// TODO: wrap Overloads in Functions
		if existing := self.AddMethod(meth.obj); existing != nil {
			// Already declared
			err := redeclaredError(meth.obj, existing, false)
			c.fileError(err, meth.obj.file)
			return
		}
		// TODO: check Function body / alias target
	}
}

// collectInitializers checks the signature of each initalizer function and
// associates them with the type with name typeName. Each [*Object] in inits
// should have type [*Overload]. The type with name typeName is looked up
// within the context and validated.
func (c *Checker) collectInitializers(ctx *Context, typeName string, inits []*Object) {
	identRange := func(obj *Object) ranges.Range {
		return c.moduleDecls[obj].node.(*ast.FunctionDeclaration).Identifier.Range()
	}
	selfObj := ctx.Lookup(typeName)
	if selfObj == nil {
		selfObj = ctx.LookupRecursive(typeName)
		if selfObj == nil {
			// Undefined
			for _, o := range inits {
				err := errors.Undefined(typeName, identRange(o))
				c.fileError(err, o.file)
			}
			return
		}
		// Found, but in a different scope
		det := []errors.Detail{{
			File:    selfObj.FilePath(),
			Range:   selfObj.Range(),
			Message: errors.Quote(typeName) + " was declared here",
		}}
		for _, o := range inits {
			node := c.moduleDecls[o].node.(*ast.FunctionDeclaration)
			err := errors.Node(errors.ErrMethodInOtherScope, node)
			err.SetParam("initializer", true)
			err.Details = det
			c.error(err)
		}
		return
	}
	switch self := selfObj.typ.(*TypeName).Type.(type) {
	case *Struct:
		self.Initializers = inits
	case *Enum:
		self.Initializers = inits
	case *TypeAlias:
		// Similar to method receivers, this can't be an alias
		for _, o := range inits {
			err := errors.RangedTypeError(errors.ErrAliasSelfType, identRange(o), nil)
			err.SetParam("initializer", true)
			err.Label = "Initializer target can't be an alias"
			c.fileError(err, o.file)
		}
		return
	default:
		// Type doesn't support initializers
		for _, o := range inits {
			err := errors.RangedTypeError(errors.ErrUnsupportedInitType, identRange(o), nil)
			err.Label = "Can't create initializers on this kind of type"
			c.fileError(err, o.file)
		}
	}
	// TODO: check Function body
}

func (c *Checker) validateReceiver(name string, self *Object,
	methods []methodInfo, isOtherScope bool,
) bool {
	selfRange := func(meth methodInfo) ranges.Range {
		if meth.decl != nil {
			return meth.decl.Struct.Range()
		} else {
			return meth.alias.Struct.Range()
		}
	}
	// Error if:
	// Object is nil
	// Object is declared in another scope
	// Object is in another module or is a builtin (TODO)
	// Object is a type alias
	// Object doesn't accept methods
	switch {
	case self == nil:
		// Self type doesn't exist
		for _, meth := range methods {
			err := errors.Undefined(name, selfRange(meth))
			c.fileError(err, meth.obj.file)
		}
	case isOtherScope:
		// typeName was declared in a different scope from the method
		det := []errors.Detail{{
			File:    self.FilePath(),
			Range:   self.Range(),
			Message: errors.Quote(name) + " was declared here",
		}}
		for _, meth := range methods {
			err := errors.Node(errors.ErrMethodInOtherScope, meth.decl)
			err.Details = det
			c.error(err)
		}
	case self.module != methods[0].obj.module:
		// TODO: check that receiver is not a primitive
		return false
	default:
		tn, ok := self.typ.(*TypeName)
		if !ok {
			panic("receiver type is not *TypeName")
			// return false
		}
		switch tn.Type.(type) {
		case *TypeAlias:
			// Self type is a type alias
			for _, m := range methods {
				err := errors.RangedTypeError(errors.ErrAliasSelfType,
					selfRange(m), nil,
				)
				err.Label = "Self type can't be an alias"
				c.fileError(err, m.obj.file)
			}
		default:
			// Self type doesn't support methods
			for _, m := range methods {
				err := errors.RangedTypeError(errors.ErrUnsupportedSelfType,
					selfRange(m), nil,
				)
				err.Label = "Can't declare methods on this kind of type"
				c.fileError(err, m.obj.file)
			}
		case MethodAdder:
			return true
		}
	}
	return false
}
