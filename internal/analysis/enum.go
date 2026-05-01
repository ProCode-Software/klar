package analysis

import "github.com/ProCode-Software/klar/internal/ast"

// Not a value.
type Enum struct {
	ItemType Type
	Items    []*EnumItem
	itemMap  map[string]*EnumItem
	Methods  []*Object // Type [*Function]
}

// Can be used as a value.
type EnumItem struct {
	Name     string
	Params   []Type
	paramMap map[string]int
	Value    any
	Enum     *Enum // For access to value type and methods
}

func (c *Checker) checkEnumDecl(o *Object, node *ast.EnumDeclaration, fileCtx *Context) {
}

func (*Enum) Kind() Kind                        { return KindEnum }
func (*Enum) String() string                    { return "" }
func (*Enum) StringWithName(name string) string { return name }

func (item *EnumItem) ParamByName(label string) Type {
	i, ok := item.paramMap[label]
	if !ok {
		return nil
	}
	return item.Params[i]
}
