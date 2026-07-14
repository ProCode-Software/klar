package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

type Struct struct {
	Inherited    map[Type]struct{}  // Structs, interfaces, and tags
	Fields       []*Object          // Type is [*StructField]
	fieldMap     map[string]*Object // Contains fields and methods
	Initializers []*Object          // Type is [*Overload]
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

type StructField struct {
	*Variable
	Optional   bool // Has default param or Optional type
	Attributes *Attributes
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

	if len(node.Fields) == 0 {
		// TODO: Remove when fmset is implemented
		str.fieldMap = make(map[string]*Object, 0)
		return
	}
	str.fieldMap = make(map[string]*Object)
	str.Fields = make([]*Object, 0, len(node.Fields))
	for _, field := range node.Fields {
		var (
			typ   = c.parseType(field.Type, fctx)
			attrs = c.parseAttributes(
				field.Attributes, attrTargetKindOf(field, true),
				field.Range, o.file,
			)
		)
		for _, id := range field.Names {
			f := &StructField{
				Variable:   &Variable{VarKind: StructFieldVar, Type: typ},
				Attributes: attrs,
			}
			obj := NewObject(id.Name, o.file, field.Range, o.module, f)
			f.Variable.Object = obj
			str.Fields = append(str.Fields, obj)
			if _, ok := str.fieldMap[id.Name]; ok {
				// Duplicate struct fields should have already been checked by the parser
				panic("field '" + id.Name + "' already exists in struct " + o.name)
			}
			str.fieldMap[id.Name] = obj
			c.queue(func() {
				// TODO: look into this again
				// The type may not be initialized by the time we initialize this struct
				if Underlying(typ) != nil {
					f.Optional = typ.Kind() == KindOptional || field.Value != nil
				}
				// TODO: default values
			}, false)
		}
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

// makeDefaultInitializers creates the default initializers for the
// underlying struct type in o.
func (c *Checker) makeDefaultInitializers(o *Object) {
}
