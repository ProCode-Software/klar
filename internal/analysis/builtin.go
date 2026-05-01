package analysis

var BuiltInContext = &Context{}

type List struct{ Elem Type }

func (l *List) Kind() Kind     { return KindList }
func (l *List) String() string { return "[" + TypeToString(l.Elem) + "]" }
