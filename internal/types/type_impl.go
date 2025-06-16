package types

func (CoreType) type_()  {}
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

func (s Struct) GetFields() map[string]Type       { return s.Fields }
func (s Struct) GetMethods() map[string]Overloads { return s.Methods }
