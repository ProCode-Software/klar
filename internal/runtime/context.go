package runtime

import (
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type TypeDeclaration = types.TypeDeclaration

type DeclType int

const (
	DeclTypeNormal DeclType = iota
	DeclTypeType
)

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
	DeclTypes        map[string]DeclType
	Declarations     map[string]*Declaration
	TypeDeclarations map[string]*TypeDeclaration
	Parent           int
}

type Exportable interface {
	Exportable_()
}

func (Declaration) Exportable_() {}

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
		DeclTypes:        make(map[string]DeclType),
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
	if _, ok := c.DeclTypes[name]; ok {
		// Already declared
		return false
	}
	c.DeclTypes[name] = DeclTypeNormal
	c.Declarations[name] = &Declaration{
		Position: pos,
		Type:     typ,
		Value:    nil,
		Constant: constant,
	}
	return true
}

// DeclareFuncType declares a function or overload in c with the value fn.
//
// If the overload exists, it returns 1 and the existing overload.
//
// If name is declared in the context but is not a function, DeclareFuncType returns
// 2 and the existing declaration.
//
// Otherwise, DeclareFuncType returns (0, nil)
func (c *Context) DeclareFuncType(name string, fn types.Function, rang ranges.Range) (err int, data any) {
	if c.DeclTypes[name] != DeclTypeNormal {
		// Declared as a type
		return 2, c.TypeDeclarations[name]
	}
	if _, ok := c.Declarations[name]; !ok {
		c.Declarations[name] = &Declaration{
			Position: rang,
			Type: types.Overloads{{
				Function: fn,
				Position: rang,
			}},
		}
		return 0, nil
	}
	if overloads, ok := c.Declarations[name].Type.(types.Overloads); !ok {
		// Already declared and not a function
		return 2, c.Declarations[name]
	} else {
		ok = overloads.Define(fn, rang)
		if !ok {
			// Overload already declared
			other, _ := overloads.Get(fn.Params)
			return 1, other
		}
		c.Declarations[name].Type = overloads
		return 0, nil
	}
}

func (c *Context) DeclareType(name string, typ types.Type, pos ranges.Range) bool {
	if _, ok := c.DeclTypes[name]; ok {
		// Already declared
		return false
	}
	c.DeclTypes[name] = DeclTypeType
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
