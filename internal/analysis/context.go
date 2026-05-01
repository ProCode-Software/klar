package analysis

import "github.com/ProCode-Software/klar/internal/errors"

type (
	DeclKind         uint8
	ContextAttribute uint8
)

type Context struct {
	index        int
	Declarations map[string]*Object
	Parent       *Context
	Children     []*Context
	Flags        Flag
	Attrs        map[ContextAttribute]any
	Used         map[string]struct{}
}

type Declaration struct {
	Kind     DeclKind
	Value    any
	Constant bool
}

func NewContext(parent *Context, flags ...Flag) *Context {
	ctx := &Context{Parent: parent, Flags: parseFlags(flags)}
	if parent != nil && parent != BuiltInContext {
		parent.Children = append(parent.Children, ctx)
		ctx.index = len(parent.Children)
	}
	return ctx
}

const (
	_ ContextAttribute = iota
	ContextFile
	firstStmtIndex
)

func (ctx *Context) setAttribute(key ContextAttribute, val any) *Context {
	if ctx.Attrs == nil {
		ctx.Attrs = make(map[ContextAttribute]any)
	}
	ctx.Attrs[key] = val
	return ctx
}

func (ctx *Context) getAttribute(key ContextAttribute) any {
	if ctx.Attrs == nil {
		return nil
	}
	return ctx.Attrs[key]
}

func (ctx *Context) Declare(obj *Object, flag ...Flag) (existing *Object) {
	flags := parseFlags(flag)
	name := obj.Name()
	ctx.initDecls()
	if existing = ctx.Declarations[name]; existing != nil {
		return
	}
	ctx.Declarations[name] = obj
	if obj.context == nil { // TODO: should this be changed?
		obj.context = ctx
	}
	_ = flags
	return nil
}

func (ctx *Context) initDecls() {
	if ctx.Declarations == nil {
		ctx.Declarations = make(map[string]*Object)
	}
}

// Lookup returns the object with the given name in the
// current context, or nil if not found.
func (ctx *Context) Lookup(name string) *Object {
	if ctx.Declarations == nil {
		return nil
	}
	return ctx.Declarations[name]
}

func (ctx *Context) LookupRecursive(name string) *Object {
	for ; ctx != nil; ctx = ctx.Parent {
		if obj := ctx.Lookup(name); obj != nil {
			return obj
		}
	}
	return nil
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
