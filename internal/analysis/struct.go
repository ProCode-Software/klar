package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Struct struct {
	Fields       []*Object          // Type is [*StructField]
	fieldMap     map[string]*Object // Contains fields and methods
	Methods      []*Object          // Type is [*Function]
	Initializers []*Object          // Type is [*Overload]
}

type StructField struct {
	*Variable
	Optional   bool // Has default param or Optional type
	Attributes *Attributes
}

func (c *Checker) checkStructDecl(o *Object, node *ast.StructDeclaration, fileCtx *Context) {
	str := &Struct{}
	o.typ.(*TypeName).Type = str
	// TODO: inherited types
	if len(node.Fields) == 0 {
		return
	}
	str.fieldMap = make(map[string]*Object)
	str.Fields = make([]*Object, 0, len(node.Fields))
	for _, fldNode := range node.Fields {
		var (
			typ   = c.parseType(fldNode.Type, fileCtx)
			attrs = c.parseAttributes(fldNode.Attributes, structFieldAttribute)
		)
		for _, id := range fldNode.Names {
			f := &StructField{
				Variable:   &Variable{VarKind: StructFieldVar, Type: typ},
				Attributes: attrs,
				Optional:   typ.Kind() == KindOptional || fldNode.Value != nil,
			}
			obj := NewObject(id.Name, o.file, fldNode.Range, o.module, f)
			f.Variable.Object = obj
			str.Fields = append(str.Fields, obj)
			if _, ok := str.fieldMap[id.Name]; ok {
				// Duplicate struct fields should have already been checked by the parser
				panic("field '" + id.Name + "' already exists in struct " + o.name)
			}
			str.fieldMap[id.Name] = obj
			// TODO: default values
		}
	}
}

// makeDefaultInitializers creates the default initializers for the
// underlying struct type in o.
func (c *Checker) makeDefaultInitializers(o *Object) {
}

func (s *Struct) AddMethod(o *Object) (existing *Object) {
	if s.fieldMap == nil {
		s.fieldMap = make(map[string]*Object)
	}
	existing = s.fieldMap[o.name]
	if existing != nil {
		return
	}
	s.fieldMap[o.name] = o
	return nil
}

func (s *Struct) Kind() Kind                        { return KindStruct }
func (s *Struct) String() string                    { return "" }
func (s *Struct) StringWithName(name string) string { return name }
