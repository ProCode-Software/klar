package analysis

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
	ContextFile ContextAttribute = iota
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
	if obj.context == nil {
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

func (ctx *Context) Lookup(name string) *Object {
	return ctx.Declarations[name]
}
