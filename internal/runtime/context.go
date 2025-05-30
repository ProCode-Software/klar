package runtime

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

type TypeDeclaration struct {
	Position lexer.Position
	Type     types.Type
}

type Declaration struct {
	Position lexer.Position
	Type     types.Type
	Value    RuntimeVal
}

type Context struct {
	Id               int
	Declarations     map[string]Declaration
	TypeDeclarations map[string]TypeDeclaration
	Parent           int
}

type ContextMap map[int]*Context

var RuntimeContexts = make(ContextMap)
var CurrContext = 0

func NewContext(parent int) *Context {
	ctx := &Context{
		Id:               CurrContext,
		Declarations:     make(map[string]Declaration),
		TypeDeclarations: make(map[string]TypeDeclaration),
		Parent:           parent,
	}
	RuntimeContexts[CurrContext] = ctx
	CurrContext++
	return ctx
}

func (c *Context) IsRoot() bool {
	return c.Id == 0
}

func (c *Context) Declare(name string, typ types.Type, pos lexer.Position) {
	if _, ok := c.Declarations[name]; ok {
		// Already declared
		return
	}
	c.Declarations[name] = Declaration{
		Position: pos,
		Type:     typ,
		Value:    nil,
	}
}

func (c *Context) Resolve(name string) RuntimeVal {
	if val, ok := c.Declarations[name]; ok {
		return val.Value
	}
	if !c.IsRoot() {
		return RuntimeContexts[c.Parent].Resolve(name)
	}
	return nil
}
