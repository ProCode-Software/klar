package analysis

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

// TODO: name hints in context
func (c *Checker) undefinedType(name string, pos ranges.Range, _ context) {
	c.Error(errors.Undefined(errors.ErrTypeUndefined, name, pos))
}

func (c *Checker) errOverloadExists(
	name string, existing types.Overload, pos ranges.Range,
) {
	err := errors.TypeError{
		Name:      existing.StringNamed(name),
		Range:     pos,
		ErrorCode: errors.ErrOverloadExists,
	}
	c.Error(err)
}

func (c *Checker) errRedeclared(
	code errors.ErrorCode,
	name string,
	newPos ranges.Range,
	newType string,
	ctx context,
) {
	var origPos ranges.Range
	var origType string
	if ctx.DeclTypes[name] == runtime.DeclTypeType {
		dec := ctx.TypeDeclarations[name]
		origPos, origType = dec.Position, typeof(dec.Type, true)
	} else {
		dec := ctx.Declarations[name]
		origPos, origType = dec.Position, typeof(dec.Type, false)
	}
	err := errors.ParseError{
		Params: errors.ErrorParams{
			"name":     name,
			"origPos":  origPos.Start,
			"origType": origType,
			"newType":  newType,
		},
		ErrorCode: code,
		Range:     newPos,
	}
	c.Error(err)
}

func typeof(t Type, isType bool) string {
	if !isType {
		switch t.(type) {
		case types.Overloads:
			return "function"
		default:
			return "variable"
		}
	}
	switch t.(type) {
	case types.Struct:
		return "struct"
	case types.Interface:
		return "interface"
	case types.Enum:
		return "enum"
	}
	return "type"
}
