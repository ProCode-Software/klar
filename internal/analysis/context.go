package analysis

type DeclKind uint8

const (
	KindVariable DeclKind = iota
	KindFunction
	KindType
)

type Context struct {
	index        int
	Declarations map[string]Declaration
	Parent       *Context
	Children     []*Context
	Flags        Flag
	Attrs        map[string]any
}

type Declaration struct {
	Kind     DeclKind
	Value    any
	Constant bool
}

func NewContext(parent *Context, flags Flag) *Context {
	ctx := &Context{Parent: parent, Flags: flags}
	if parent != nil && parent != BuiltInContext {
		parent.Children = append(parent.Children, ctx)
		ctx.index = len(parent.Children)
	}
	return ctx
}
