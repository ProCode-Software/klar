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
	// discardDecls []*Object // Declarations with '_' name and can't be put in Declarations
	Parent   *Context
	Children []*Context
	Flags    Flag
	Attrs    map[ContextAttribute]any
	Used     map[*Object]struct{}
	File     FileID
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
	unusedReported
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

func (ctx *Context) Declare(obj *Object) (existing *Object) {
	name := obj.Name
	obj.Context = ctx
	if ctx.Declarations == nil {
		ctx.Declarations = make(map[string]*Object)
	}
	if existing = ctx.Declarations[name]; existing != nil {
		return
	}
	ctx.Declarations[name] = obj
	// TODO: Change obj's order
	return nil
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

func (c *Checker) declare(ctx *Context, obj *Object) {
	if obj.Name == "_" {
		return
	}
	if existing := ctx.Declare(obj); existing != nil {
		// Declared already
		err := redeclaredError(obj, existing, true)
		c.fileError(err, obj.File)
	}
}

func (ctx *Context) String() string {
	if ctx.Declarations == nil {
		return fmt.Sprintf("Context (file %d) {}", ctx.File)
	}
	var b strings.Builder
	var longestName int
	for _, o := range ctx.SortedDecls() {
		if len(o.Name) > longestName {
			longestName = len(o.Name)
		}
	}
	fmt.Fprintf(&b, "Context (file %v) {\n", ctx.SortedDecls()[0].FilePath())
	for _, o := range ctx.SortedDecls() {
		var typeStr any
		switch {
		case !o.IsTypeName():
			typeStr = o.Type
		case o.Type.Underlying() == nil:
			typeStr = "type = <incomplete>"
		default:
			typeStr = fmt.Sprintf("type = %v", o.Type.Underlying())
		}
		pad := strings.Repeat(" ", longestName-len(o.Name))
		fmt.Fprintf(&b, "  %s:%s %s (%s)\n", o.Name, pad, typeStr, o.Range)
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func (ctx *Context) IsEmpty() bool {
	return ctx == nil || len(ctx.Declarations) == 0
}

func (ctx *Context) markUsed(o *Object) {
	if ctx.File.Builtin() {
		return
	}
	if ctx.Used == nil {
		ctx.Used = make(map[*Object]struct{})
	}
	ctx.Used[o] = struct{}{}
}

// Unused is an iterator that yields the unused objects declared in the
// context. Public objects and objects prefixed with '_' are not yielded.
// Objects in a different module may be yielded.
func (ctx *Context) Unused(yield func(*Object) bool) {
	for _, o := range ctx.SortedDecls() {
		if _, ok := ctx.Used[o]; ok {
			continue
		}
		// TODO: Don't yield objects from different modules.
		// Store the module in the context.
		needUsage := !o.Public && !strings.HasPrefix(o.Name, "_") &&
			!o.Flags.Has(ImplicitVar|UsageOptional)
		// Only specific kinds of variables need t0o be used
		if vr, ok := o.Type.(*Variable); ok && needUsage {
			k := vr.VarKind
			needUsage = k == LocalVar || k == TopLevelVar || k == FuncParamVar
		}
		if needUsage && !yield(o) {
			return
		}
	}
}
