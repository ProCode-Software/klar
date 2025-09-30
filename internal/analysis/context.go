package analysis

type ContextID uint16

type DeclKind uint8

const (
	KindVariable DeclKind = iota
	KindFunction
	KindType
)

type Context struct {
	Id ContextID
	Declarations map[string]Declaration
	Parent ContextID
}

type Declaration struct {
	Kind DeclKind
	Value any
	Constant bool
}