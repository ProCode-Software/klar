package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Interface struct {
	// Doesn't guarantee compatibility because items can be overidden.
	// Guaranteed compatibility for tag keys.
	Inherited map[Type]struct{}

	ItemAttrs       map[string]*Attributes // Attributes for each field/method
	DeclaredFields  map[string]Type        // Explicitly declared, not inherited
	DeclaredMethods map[string]*Function   // Methods from the interface declaration
	order           []string               // Field and method order, as declared in the source
	fmset           *FieldMethodSet        // Lazy-computed

	MethodSet // Extension methods
}

func (*Interface) Kind() Kind     { return KindInterface }
func (*Interface) String() string { return "<interface>" }

func (c *Checker) checkInterfaceDecl(o *Object, decl *ast.InterfaceDeclaration) {
	fctx := o.LookupContext()
	intf := &Interface{
		Inherited:       c.checkInheritedTypes(decl.InheritedTypes, KindInterface, fctx),
		order:           make([]string, 0, len(decl.Items)),
		DeclaredFields:  make(map[string]Type),
		DeclaredMethods: make(map[string]*Function),
	}
	for _, entry := range decl.Items {
		// Attributes
		attrs := c.parseAttributes(
			entry.Attributes, attrTargetKindOf(entry, true),
			entry.Range, o.file,
		)
		if attrs != nil && intf.ItemAttrs == nil {
			intf.ItemAttrs = make(map[string]*Attributes)
		}

		// Method. Redeclared items are checked by the parser
		if meth, ok := entry.Value.(*ast.MethodType); ok {
			name := entry.Keys[0].Name // Only 1 key, validated by parser
			intf.order = append(intf.order, name)
			ov := c.checkIntfMethod(entry.Keys[0], meth, fctx)
			if par, ok := intf.DeclaredMethods[name]; ok {
				par.Overloads = append(par.Overloads, ov) // Another overload
			} else {
				// First overload
				intf.DeclaredMethods[name] = &Function{Overloads: []*Overload{ov}}
			}
			if attrs != nil {
				intf.ItemAttrs[name] = attrs
			}
			continue
		}

		// Field
		typ := c.parseType(entry.Value, fctx)
		for _, key := range entry.Keys {
			name := key.Name
			intf.order = append(intf.order, name)
			intf.DeclaredFields[name] = typ
			if attrs != nil {
				intf.ItemAttrs[name] = attrs
			}
		}
	}
	o.TypeName().Type = intf
}

func (c *Checker) checkIntfMethod(
	ident ast.Identifier, meth *ast.MethodType, fctx *Context,
) *Overload {
	return nil
}
