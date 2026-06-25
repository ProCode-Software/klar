package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type DeclarationInfo struct {
	node     ast.Statement
	varInfo  *varInfo // For var/const declaration.
	funcKind funcKind // For function declaration.
	receiver *Object  // For method or initializer. Should be [*TypeName]
}

type varInfo struct {
	lhs     ast.Destructurable // Where the variable was defined. Undestructured
	rhs     ast.Expression     // Undestructured RHS expression
	expType Type               // If the decl has an explicit type. Use as [*Expr.hint]
	// Set when rhs is checked, or the explicit type is known. Pointer is
	// never nil, but the *Expr can be.
	rhsExpr **Expr
}

type funcKind uint8

const (
	normalFunc funcKind = iota
	methodFunc
	initFunc
)

// declareWithInfo declares an object into the given context with the
// given attributes and [DeclarationInfo]. If declareToCtx == false,
// only the object's information is recorded and is not added to the context.
// If *attrs != nil, the attributes are parsed and drained.
func (c *Checker) declareWithInfo(obj *Object, ctx *Context,
	attrs *[]*ast.Attribute, declareToCtx bool,
) {
	if declareToCtx {
		c.declare(ctx, obj)
	} else {
		obj.context = ctx // Still set the object's context
	}
	// Parse the attributes if any (top-level only)
	if attrs != nil {
		obj.attrs = c.parseAttributes(*attrs, attrTargetKindOf(obj.info.node), obj.file)
		*attrs = (*attrs)[:0]
	}
	// obj.order = uint32(len(ctx.declInfo))
}

func (c *Checker) checkDeclaration(o *Object) {
	/*
		Red: Type isn't known yet. Not in objPathIndex.
		White: Type is pending. Is in objPathIndex.
		Blue: Type is known. Not in objPathIndex.

		Blue can only depend on blue. White/grey can only depend on red or blue.
		A dependency on white is a (possibly invalid) cycle.

		When marked white, it is pushed onto the object path stack, and its index
		is recorded in objPathIndex. It's removed from the map and the stack when marked blue.
	*/
	if _, ok := c.objPathIndex[o]; ok {
		switch typ := o.typ.(type) {
		case *Variable:
			if !c.isValidCycle(o) || typ.Type == nil {
				typ.Type = InvalidType
			}
		case *Constant:
			if !c.isValidCycle(o) || typ.Type == nil {
				typ.Type = InvalidType
			}
		case *TypeName:
			if !c.isValidCycle(o) {
				typ.Type = InvalidType
			}
		case *Function, *Overload:
			c.isValidCycle(o) // TODO: is this needed?
		case *FunctionAlias:
		// TODO
		default:
			panic(fmt.Sprintf("unhandled declaration type: %T", o.typ))
		}
		if o.typ.Underlying() == nil {
			panic("underlying type is still nil")
		}
		return
	}

	if o.typ.Underlying() != nil {
		return // Blue, already checked
	}

	// White, not checked yet
	c.pushToPath(o)
	defer c.popPath()

	switch o.typ.(type) {
	case *Variable:
		c.checkVarDecl(o)
	case *Constant:
		c.checkConstDecl(o)
	case *TypeName:
		c.checkTypeDecl(o)
	case *Function:
		c.checkFuncDecl(o)
	case *Overload:
		return // Overloads are part of functions
	case *FunctionAlias:
		c.checkFuncAlias(o)
	default:
		panic(fmt.Sprintf("unhandled declaration type: %T", o.typ))
	}
}

// An error is reported if isValidCycle returns false.
func (c *Checker) isValidCycle(o *Object) bool {
	start := c.objPathIndex[o]
	cycle := c.objPath[start:]
	// Number of type defs and values (const or var) in the cycle
	var typeDefCount, valCount int
	for _, obj := range cycle {
		switch typ := obj.typ.(type) {
		case *TypeName:
			if _, ok := typ.Type.(*TypeAlias); !ok {
				// Only increase the count for non-aliases
				typeDefCount++
			}
		case *Variable, *Constant:
			valCount++
		case *Function:
		default:
			panic(fmt.Sprintf("isValidCycle: unhandled declaration type: %T", obj.typ))
		}
	}
	switch {
	case valCount == len(cycle):
		// Go: A cycle involving only constants and variables is invalid but we
		// ignore them here because they are reported via the initialization
		// cycle check.
		return true
	case valCount == 0 && typeDefCount > 0:
		// A cycle involving only type definitions (and maybe functions) must have
		// at least 1 type definition to be valid. Alias-only cycles are invalid.
		return true
	default:
		// Invalid cycle
		c.error(cycleError(cycle))
		return false
	}
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
	self := Underlying(selfObj.typ).(SupportsMethods)
	for _, meth := range methods {
		meth.obj.info.funcKind = methodFunc
		meth.obj.info.receiver = selfObj
		if err := self.AddMethod(meth.obj); err != nil {
			if err.Code == klarerrs.ErrFieldAndMethodSameName {
				err.SetParam("type", typeName)
			}
			c.fileError(err, meth.obj.file)
			return
		}
	}
	// Typecheck the [*Function] or [*FunctionAlias] objects, not overloads from `methods`
	for _, obj := range self.GetMethods() {
		c.checkDeclaration(obj)
	}
}

// collectInitializers checks the signature of each initializer function and
// associates them with the type with name typeName. Each [*Object] in inits
// should have type [*Overload]. The type with name typeName is looked up
// within the context and validated.
func (c *Checker) collectInitializers(ctx *Context, typeName string, inits []*Object) {
	identRange := func(obj *Object) ranges.Range {
		return obj.info.node.(*ast.FunctionDeclaration).Identifier.Range()
	}
	selfObj := ctx.Lookup(typeName)
	if selfObj == nil {
		selfObj = ctx.LookupRecursive(typeName)
		if selfObj == nil {
			// Undefined
			for _, o := range inits {
				err := klarerrs.Undefined(typeName, identRange(o))
				c.fileError(err, o.file)
			}
			return
		}
		// Found, but in a different scope
		det := []klarerrs.Detail{{
			File:    selfObj.FilePath(),
			Range:   selfObj.Range(),
			Message: klarerrs.Quote(typeName) + " was declared here",
		}}
		for _, o := range inits {
			node := o.info.node.(*ast.FunctionDeclaration)
			err := klarerrs.Node(klarerrs.ErrMethodInOtherScope, node)
			err.SetParam("initializer", true)
			err.Details = det
			c.fileError(err, o.file)
		}
		return
	}
	switch self := selfObj.TypeName().Type.(type) {
	case *Struct:
		self.Initializers = inits
	case *Enum:
		self.Initializers = inits
	case *TypeAlias:
		// Similar to method receivers, this can't be an alias
		for _, o := range inits {
			err := klarerrs.Range(klarerrs.ErrAliasSelfType, identRange(o))
			err.SetParam("initializer", true)
			err.Label = "Initializer target can't be an alias"
			c.fileError(err, o.file)
		}
		return
	default:
		// Type doesn't support initializers
		for _, o := range inits {
			err := klarerrs.Range(klarerrs.ErrUnsupportedInitType, identRange(o))
			err.Label = "Can't create initializers on this kind of type"
			c.fileError(err, o.file)
		}
		return
	}
	for _, obj := range inits {
		obj.info.funcKind = initFunc
		obj.info.receiver = selfObj
		c.checkOverload(obj.typ.(*Overload), nil)
	}
}

func (c *Checker) validateReceiver(name string, self *Object,
	methods []methodInfo, isOtherScope bool,
) bool {
	selfRange := func(meth methodInfo) ranges.Range {
		if meth.decl != nil {
			return meth.decl.SelfType.Range()
		} else {
			return meth.alias.Struct.Range()
		}
	}
	// Error if:
	// - Object is nil
	// - Object is not a type
	// - Object is declared in another scope
	// - Object is in another module or is a builtin (TODO)
	// - Object is a type alias
	// - Object doesn't accept methods
	switch {
	case self == nil:
		// Self type doesn't exist
		for _, meth := range methods {
			err := klarerrs.Undefined(name, selfRange(meth))
			c.fileError(err, meth.obj.file)
		}
	case isOtherScope:
		// typeName was declared in a different scope from the method
		det := []klarerrs.Detail{{
			File:    self.FilePath(),
			Range:   self.Range(),
			Message: klarerrs.Quote(name) + " was declared here",
		}}
		for _, meth := range methods {
			err := klarerrs.Node(klarerrs.ErrMethodInOtherScope, meth.decl)
			err.Details = det
			c.fileError(err, meth.obj.file)
		}
	case self.module != methods[0].obj.module:
		// TODO: check that receiver is not a primitive
		return false
	default:
		tn, ok := self.typ.(*TypeName)
		if !ok {
			// Receiver is not a type
			for _, m := range methods {
				err := klarerrs.Range(klarerrs.ErrUnsupportedSelfType, selfRange(m))
				err.Label = "This isn't a type"
				c.fileError(err, m.obj.file)
			}
			return false
		}
		switch tn.Type.(type) {
		case *TypeAlias:
			// Self type is a type alias
			for _, m := range methods {
				err := klarerrs.Range(klarerrs.ErrAliasSelfType, selfRange(m))
				err.Label = "Self type can't be an alias"
				c.fileError(err, m.obj.file)
			}
		default:
			// Self type doesn't support methods
			for _, m := range methods {
				err := klarerrs.Range(klarerrs.ErrUnsupportedSelfType, selfRange(m))
				err.Label = "Can't declare methods on this kind of type"
				c.fileError(err, m.obj.file)
			}
		case SupportsMethods:
			return true
		}
	}
	return false
}
