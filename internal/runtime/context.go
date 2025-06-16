package runtime

import (
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type TypeDeclaration = types.TypeDeclaration

type Declaration struct {
	Position   ranges.Range
	Type       types.Type
	Constant   bool
	Value      RuntimeVal
	Used       bool
	Attributes map[string]any
}

type Context struct {
	Id               int
	Declarations     map[string]*Declaration
	TypeDeclarations map[string]*TypeDeclaration
	Parent           int
}

type Exportable interface {
	Exportable_()
}

func (Declaration) Exportable_()     {}

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
	name string, constant bool, typ types.Type, pos ranges.Range,
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

func (c *Context) DeclareType(name string, typ types.Type, pos ranges.Range) bool {
	if _, ok := c.TypeDeclarations[name]; ok {
		// Already declared
		return false
	}
	c.TypeDeclarations[name] = &TypeDeclaration{
		Type:     typ,
		Position: pos,
	}
	return true
}

func (c *Context) Resolve(name string) (d *Declaration, found bool) {
	if val, ok := c.Declarations[name]; ok {
		val.Used = true
		return val, true
	}
	if c.Parent > -1 {
		return RuntimeContexts[c.Parent].Resolve(name)
	}
	return nil, false
}

func (c *Context) ResolveType(name string) (d *TypeDeclaration, found bool) {
	if val, ok := c.TypeDeclarations[name]; ok {
		val.Used = true
		return val, true
	}
	if c.Parent > -1 {
		return RuntimeContexts[c.Parent].ResolveType(name)
	}
	return nil, false
}

func (c *Context) SetType(name string, typ types.Type) (success bool) {
	resolved, ok := c.ResolveType(name)
	if !ok {
		return false
	}
	resolved.Type = typ
	return true
}

// GetContext returns the context with id. If the context does not exist, GetContext
// returns nil.
func GetContext(id int) *Context {
	return RuntimeContexts[id]
}
