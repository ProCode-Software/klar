package klarml

import "fmt"

type Context struct {
	Document   Document
	Namespaces map[string]Resolver
}
type Resolver func(input Node) Node

func NewContext(doc Document) *Context {
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
	vars := make(map[string]Value)
	var currentNs string
	doc, _ := Walk(c.Document, func(n *Node) (Node, error) {
		switch n := (*n).(type) {
		case VarDecl:
			vars[n.Name] = n.Value
		case VarRef:
			if value, ok := vars[n.Identifier]; ok {
				return value, nil
			}
			errors = append(errors, fmt.Errorf("variable '%s' is not defined", n.Identifier))
			return nil, nil
		case Namespace:
			currentNs = n.Name
		default:
			if currentNs != "" {
				handler := c.Namespaces[currentNs]
				handler(n)
				currentNs = ""	
			}
		}
		return *n, nil
	})
	c.Document = doc
	return nil
}
