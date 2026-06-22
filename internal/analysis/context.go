package analysis

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type (
	DeclKind         uint8
	ContextAttribute uint8
)

type Context struct {
	index        int
	Declarations map[string]*Object
	sortedDecls  []*Object // By object order. Lazily sorted; never reference directly.
	Parent       *Context
	Children     []*Context
	Flags        Flag
	Attrs        map[ContextAttribute]any
	Used         map[string]struct{}
	File         FileID
}

type Declaration struct {
	Kind     DeclKind
	Value    any
	Constant bool
}

func NewContext(parent *Context, fid FileID, flags ...Flag) *Context {
	ctx := &Context{Parent: parent, Flags: parseFlags(flags), File: fid}
	if parent != nil && parent != BuiltInContext {
		parent.Children = append(parent.Children, ctx)
		ctx.index = len(parent.Children)
	}
	return ctx
}

const (
	_ ContextAttribute = iota
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

func (ctx *Context) SortedDecls() []*Object {
	if ctx.sortedDecls == nil {
		ctx.sortedDecls = slices.SortedFunc(maps.Values(ctx.Declarations), sortByOrder)
	}
	return ctx.sortedDecls
}

func (c *Checker) declare(ctx *Context, obj *Object, flags ...Flag) {
	if obj.name == "_" {
		return
	}
	if existing := ctx.Declare(obj, flags...); existing != nil {
		// Declared already
		err := redeclaredError(obj, existing, true)
		c.fileError(err, obj.file)
	}
}

func (ctx *Context) String() string {
	if ctx.Declarations == nil {
		return fmt.Sprintf("Context (file %d) {}", ctx.File)
	}
	var b strings.Builder
	var longestName int
	for _, o := range ctx.SortedDecls() {
		if len(o.Name()) > longestName {
			longestName = len(o.Name())
		}
	}
	fmt.Fprintf(&b, "Context (file %v) {\n", ctx.SortedDecls()[0].FilePath())
	for _, o := range ctx.SortedDecls() {
		var typeStr any
		switch {
		case !o.IsTypeName():
			typeStr = o.typ
		case o.typ.Underlying() == nil:
			typeStr = "type = <incomplete>"
		default:
			typeStr = fmt.Sprintf("type = %v", o.typ.Underlying())
		}
		pad := strings.Repeat(" ", longestName-len(o.Name()))
		fmt.Fprintf(&b, "  %s:%s %s (%s)\n", o.Name(), pad, typeStr, o.rang)
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}
