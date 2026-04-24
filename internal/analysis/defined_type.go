package analysis

type DefinedType struct {
	*Object
	Methods map[string]*Function
}

func (t *DefinedType) AddMethod(m *Object) {
	t.Unpack()
	if _, ok := t.Methods[m.name]; !ok {
		t.Methods[m.name] = m.typ.(*Function)
	}
}

func (t *DefinedType) Unpack() {}
