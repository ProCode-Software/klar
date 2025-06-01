package runtime

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

type TypeDeclaration struct {
	Type types.Type
	Used bool
}

type Declaration struct {
	Position lexer.Position
	Type     types.Type
	Constant bool
	Value    RuntimeVal
	Used     bool
}

type Context struct {
	Id               int
	Declarations     map[string]*Declaration
	TypeDeclarations map[string]*TypeDeclaration
	Parent           int
}

type Exportable interface {
	Exportable()
}

func (Declaration) Exportable()     {}
func (TypeDeclaration) Exportable() {}

type ContextMap map[int]*Context

var (
	RuntimeContexts = make(ContextMap)
	CurrContext     = 0
)

// NewContext creates a new context inside the program. If parent is -1,
// there is no parent, indicating that this is the root context.
func NewContext(parent int) *Context {
	ctx := &Context{
		Id:               CurrContext,
		Declarations:     make(map[string]*Declaration),
		TypeDeclarations: make(map[string]*TypeDeclaration),
		Parent:           parent,
	}
	RuntimeContexts[CurrContext] = ctx
	CurrContext++
	return ctx
}

func (c *Context) IsRoot() bool {
	return c.Id == 0
}

// Declare declares a new variable. If the variable already exists, Declare returns false.
func (c *Context) Declare(
	name string, constant bool, typ types.Type, pos lexer.Position,
) bool {
	if _, ok := c.Declarations[name]; ok {
		// Already declared
		return false
	}
	c.Declarations[name] = &Declaration{
		Position: pos,
		Type:     typ,
		Value:    nil,
		Constant: constant,
	}
	return true
}

func (c *Context) DeclareType(name string, typ types.Type) bool {
	if _, ok := c.TypeDeclarations[name]; ok {
		// Already declared
		return false
	}
	c.TypeDeclarations[name] = &TypeDeclaration{
		Type: typ,
	}
	return true
}

func (c *Context) Resolve(name string) (d *Declaration, found bool) {
	if val, ok := c.Declarations[name]; ok {
		val.Used = true
		return val, true
	}
	if !c.IsRoot() {
		return RuntimeContexts[c.Parent].Resolve(name)
	}
	return nil, false
}

// GetContext returns the context with id. If the context does not exist, GetContext
// returns nil.
func GetContext(id int) *Context {
	return RuntimeContexts[id]
}
