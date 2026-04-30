package analysis

type MethodAdder interface {
	// Method returns the method with the given name, or nil if it doesn't exist.
	// Method(name string) *Function

	// AddMethod adds the method m to the type. If a method with the same name
	// already exists on the type, the existing method is returned instead.
	// m should have type [*Function], however, existing's type may not be [*Function].
	AddMethod(m *Object) (existing *Object)
}

type Struct struct {
	Fields   []*Object          // Type is [*StructField]
	fieldMap map[string]*Object // Contains fields and methods
	Methods  []*Object          // Type is [*Function]
}

type StructField struct {
	*Variable
	Optional   bool // Has default param or Optional type
	Attributes *Attributes
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
