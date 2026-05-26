package klon

import "github.com/ProCode-Software/klar/pkg/klon/klonerrs"

type ClassDirection int

const (
	LeftClass      ClassDirection = iota // Arguments on left
	RightClass                           // Arguments on right
	LeftRightClass                       // Arguments on both sides
	VoidClass                            // No arguments
)

type Class struct {
	Name      string
	Direction ClassDirection
	// TODO: transform func
}

type Context struct {
	Classes map[string]Class
	// Map of field IDs to their enum options
	Enums map[string]map[string]any
	// If Warnings != nil, warnings will be appended here
	Warnings []*Error
	// Error codes that should be treated as warnings instead of errors
	WarningKinds map[klonerrs.Code]struct{}
}

// SetWarningKinds sets the error codes that should be treated as warnings instead of errors
func (c *Context) SetWarningKinds(codes ...klonerrs.Code) {
	if c.Warnings == nil {
		c.Warnings = make([]*Error, 0)
	}
	if c.WarningKinds == nil {
		c.WarningKinds = make(map[klonerrs.Code]struct{}, len(codes))
	}
	for _, code := range codes {
		c.WarningKinds[code] = struct{}{}
	}
}
