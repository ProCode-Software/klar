package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Interface struct {
	Inherited map[Type]struct{}

	DeclaredFields  map[string]Type      // Explicitly declared, not inherited
	DeclaredMethods map[string]*Function // Methods from the interface declaration
	order           []string             // Field and method order, as declared in the source
	fmset           *FieldMethodSet      // Lazy-computed

	MethodSet // Extension methods
}

func (*Interface) Kind() Kind { return KindInterface }

func (c *Checker) checkInterfaceDecl(
	o *Object, decl *ast.InterfaceDeclaration, fctx *Context,
) {
	intf := &Interface{
		Inherited: c.checkInheritedTypes(
			decl.InheritedTypes, KindInterface, o.file, fctx,
		),
		order:           make([]string, 0, len(decl.Items)),
		DeclaredFields:  make(map[string]Type),
		DeclaredMethods: make(map[string]*Function),
	}
	for _, entry := range decl.Items {
		attrs := c.parseAttributes(entry.Attributes, intfFieldAttribute, o.file)
		if meth, ok := entry.Value.(*ast.MethodType); ok {
			name := entry.Keys[0].Name
			ov := c.checkIntfMethod(entry.Keys[0], meth, fctx)
			_ = ov
			_ = name
		}
		typ := c.parseType(entry.Value, fctx)
		for _, key := range entry.Keys {
			name := key.Name
			intf.order = append(intf.order, name)
		}
		_, _ = typ, attrs

	}
	o.TypeName().Type = intf
}

func (c *Checker) checkIntfMethod(ident ast.Identifier, meth *ast.MethodType, fctx *Context) *Overload {
	return nil
}
