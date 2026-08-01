package analysis

import (
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Struct struct {
	Inherited    map[Type]struct{}  // Structs, interfaces, and tags
	Fields       []*Object          // Type is [*Variable]
	fieldMap     map[string]*Object // Contains fields and methods
	Initializers []*Overload        // Type is [*Overload]
	MethodSet
	fmset *FieldMethodSet // Lazy-computed
}

var _ SupportsMethods = &Struct{}

func (s *Struct) String() string { return "<struct>" }
func (s *Struct) Kind() Kind     { return KindStruct }

type FieldMethodSet struct {
	All     map[string]Type
	Fields  map[string]Type
	Methods map[string]*Function
	// TODO: Should we add a map of ambiguous field/methods?
}

// checkStructDecl checks a struct declaration and sets o's underlying type
// to a [*Struct]. o's Type should be [*TypeName].
func (c *Checker) checkStructDecl(o *Object, node *ast.StructDeclaration) {
	str := &Struct{}
	str.nonMethodMap = &str.fieldMap
	fctx := o.LookupContext()
	o.TypeName().Type = str

	// We're just checking their kinds for now. TODO: Add the fields and methods later.
	str.Inherited = c.checkInheritedTypes(node.InheritedTypes, KindStruct, fctx)

	if len(node.Fields) > 0 {
		str.fieldMap = make(map[string]*Object)
		str.Fields = make([]*Object, 0, len(node.Fields))
	}
	for _, fnode := range node.Fields {
		typ := c.parseType(fnode.Type, fctx)
		attrs := c.parseAttributes(
			fnode.Attributes, attrTargetKindOf(fnode, true), fnode.Range, o.File,
		)
		for _, id := range fnode.Names {
			f := NewObject(id.Name, o.File, fnode.Range, o.Module, nil)
			NewVariable(f, StructFieldVar, typ)
			f.attrs = attrs
			str.Fields = append(str.Fields, f)
			if _, ok := str.fieldMap[id.Name]; ok {
				// Duplicate struct fields should have already been checked by the parser
				panic("field '" + id.Name + "' already exists in struct " + o.Name)
			}
			str.fieldMap[id.Name] = f

			// Check the default value
			if fnode.Value != nil {
				f.Flags |= HasDefault
				// Check that the default value matches the type
				c.queue(func() {
					e := c.checkExpr(fnode.Value, NewExpr(fctx).withHint(typ))
					if !Compatible(e.Type, typ) {
						err := typeMismatch(typ, e.Type, fnode.Value.GetRange())
						err.AddHighlight(
							"The field has type "+typ.String(), fnode.Type.GetRange(),
						)
						c.fileError(err, o.File)
					}
				}, false)
			}
		}
	}

	// Once the struct's methods are attached, check that the struct
	// implements any inherited interfaces
	if len(str.Inherited) > 0 {
		c.queue(func() {
			for inh := range str.Inherited {
				if inh.Kind() != KindInterface || Implements(str, inh) {
					continue
				}
				// TODO: Find range at which the interface is inherited and report error
			}
		}, true)
	}
}

func (s *Struct) Index(f string, t *Expr) *klarerrs.Error {
	// TODO: use fmset to also add inherited fields/methods
	if obj, ok := s.fieldMap[f]; ok {
		t.Type = obj
		return nil
	}
	err := fieldNotFound(f)
	if len(s.fieldMap) == 0 {
		err.Hint("The struct has no fields.")
	}
	return err
}

// IsOptionalParam returns true if the provided function parameter
// or struct field can be omitted.
func IsOptionalParam(p *Object) bool {
	return p.Kind() == KindOptional || p.Flags.Has(HasDefault)
}

// tryCheckInitializer attempts to resolve an initializer from inits. If
// successful, the initialized type is returned, possibly wrapped in an optional
// or result, following the rules of custom initializers. The inferred parameters
// are returned for use in default initializer checking.
func (c *Checker) tryCheckInitializer(
	inits []*Overload, ps paramSet, args []*ast.CallParam, t *Expr,
) bool {
	if len(inits) == 0 {
		return false
	}
	ov, exact, err := c.resolveOverload(inits, ps, true)
	if !exact {
		return false
	}
	if err != nil {
		// Idk if this should happen
		panic("exact overload found, but resolveOverload returned a warning")
	}
	c.checkOverloadParams(ov, ps, args, t)
	return true
}

// If no initializer is found, checkDefaultStructInit returns false if exactly
// 1 parameter is provided, or reports an error and returns true.
func (c *Checker) checkDefaultStructInit(s *Struct,
	args []*ast.CallParam, ps paramSet, fid FileID,
) bool {
	// Keys are field indices. This is to ensure a field isn't provided as
	// a positional param and a labelled param:
	// 	type Person { name: String, age: Int }
	//  Person("John", 32, age: 32) // 'age' provided already
	providedFields := make(map[*Object]struct{}, len(s.Fields))
	checkType := func(exp *Object, got Type, node ast.Expression) {
		if !Compatible(got, exp.Type) {
		}
	}
	// Check labelled params first
	for i, param := range ps.params {
		node := ps.nodeMap[i]
		if i > len(s.Fields) {
			// Too many fields provided
			firstLabelled := slices.IndexFunc(args, func(p *ast.CallParam) bool {
				return p.Label != nil
			})
			if firstLabelled < 0 {
				firstLabelled = len(args) - 1
			}
			err := klarerrs.Range(klarerrs.ErrTooManyInitFields, ranges.Between(
				node.GetRange(), args[firstLabelled].GetRange(),
			))
			err.Label = "Extra parameters"
			c.fileError(err, fid)
			break
		}
		field := s.Fields[i]
		// If this is the only parameter and it doesn't match, return early
		// instead of reporting an error. This is a type cast.
		if i == 0 && len(ps.params) == 1 && ps.NumLabelled() == 0 &&
			!Compatible(param, field.Type) {
			return false
		}
		checkType(field, param, node)
		providedFields[field] = struct{}{}
	}
	// Labelled params
	for name, param := range ps.labelled {
		field, ok := s.fieldMap[name]
		node := args[slices.IndexFunc(args, func(a *ast.CallParam) bool {
			return a.Label != nil && a.Label.Name == name
		})]
		if !ok {
			// Field not found
			err := klarerrs.Node(klarerrs.ErrFieldNotFound, node.Label)
			err.Name = name
			err.Desc = "The struct has these fields: " + strings.Join(
				slices.Sorted(maps.Keys(s.fieldMap)), ", ",
			)
			c.fileError(err, fid)
			continue
		}
		checkType(field, param, node.Value)
		if _, ok := providedFields[field]; ok {
			err := klarerrs.Node(klarerrs.ErrPositionalFieldProvided, node.Label)
			err.Name = name
			positional := ps.nodeMap[slices.Index(s.Fields, field)]
			err.AddHighlight(
				"This field was already provided without a label",
				positional.GetRange(),
			)
			c.fileError(err, fid)
		}
		providedFields[field] = struct{}{}
	}

	// Check for missing fields
	for _, field := range s.Fields {
		if _, ok := providedFields[field]; ok || IsOptionalParam(field) {
		}
	}

	// TODO: Report an error that there is no valid initializer (default
	// or custom) for the provided params. List all initializers.

	// TODO: Ensure there are no variadic rests
	return true
}
