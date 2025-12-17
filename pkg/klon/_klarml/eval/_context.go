package klon

import (
	"fmt"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

type Context struct {
	Document   *ast.Document
	Namespaces map[string]Resolver
}
type Resolver func(input ast.Node) ast.Node

func NewContext(doc *ast.Document) *Context {
	return &Context{
		Document:   doc,
		Namespaces: nil,
	}
}

func (c *Context) DefineNamespace(name string, r Resolver) {
	if r == nil {
		panic("c.DefineNamespace(name, r): r is a nil function")
	}
	if _, ok := c.Namespaces[name]; ok {
		panic("c.DefineNamespace(" + name + ", r): namespace already defined in c")
	}
	c.Namespaces[name] = r
}

func (c *Context) ResolveVars() (errors []error) {
	vars := make(map[string]ast.Value)
	var currentNs string
	err := ast.Walk(c.Document, func(n ast.Node) error {
		switch n := n.(type) {
		case *ast.VarDecl:
			vars[n.Name] = n.Value
		case *ast.VarRef:
			if value, ok := vars[n.Identifier]; ok {
				return nil
			}
			errors = append(errors, fmt.Errorf("variable '%s' is not defined", n.Identifier))
			return nil
		case *ast.Namespace:
			currentNs = n.Name
		default:
			if currentNs != "" {
				handler := c.Namespaces[currentNs]
				handler(n)
				currentNs = ""
			}
		}
		return nil
	})
	c.Document = doc
	return nil
}
