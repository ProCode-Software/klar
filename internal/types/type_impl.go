package types

func (CoreType) type_()  {}
func (Untyped) type_()   {}
func (Enum) type_()      {}
func (Generic) type_()   {}
func (Lambda) type_()    {}
func (List) type_()      {}
func (Map) type_()       {}
func (Optional) type_()  {}
func (Ref) type_()       {}
func (Result) type_()    {}
func (Struct) type_()    {}
func (Tuple) type_()     {}
func (Union) type_()     {}
func (Overloads) type_() {}

func (s Struct) GetFields() FieldMap   { return s.Fields }
func (s Struct) GetMethods() MethodMap { return s.Methods }

func (e Enum) GetFields() FieldMap {
	fields := make(FieldMap, len(e.Members))
	valueType := e.ValueType
	for name := range e.Members {
		fields[name] = valueType
	}
	return fields
}
func (e Enum) GetMethods() MethodMap { return nil }
