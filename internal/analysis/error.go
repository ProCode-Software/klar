package analysis

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

// TODO: name hints in context
func (c *Checker) ErrUndefinedType(name string, pos ranges.Range, ctx context) {
	c.undefined(name, pos, true, ctx)
}

func (c *Checker) ErrUndefinedVar(name string, pos ranges.Range, ctx context) {
	c.undefined(name, pos, false, ctx)
}

func (c *Checker) ErrNotInEnum(name string, pos ranges.Range, enum types.Enum) {
}

func (c *Checker) undefined(name string, pos ranges.Range, isType bool, _ context) {
	code := errors.ErrVarUndefined
	if isType {
		code = errors.ErrTypeUndefined
	}
	c.Error(errors.Undefined(code, name, pos))
}

func (c *Checker) ErrOverloadExists(
	name string, existing types.Overload, pos ranges.Range,
) {
	err := errors.TypeError{
		Name:      existing.StringNamed(name),
		Range:     pos,
		ErrorCode: errors.ErrOverloadExists,
		Params: errors.ErrorParams{
			"origPos": existing.Position,
		},
	}
	c.Error(err)
}

func (c *Checker) ErrRedeclared(
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

func (c *Checker) TypeMismatch(code errors.ErrorCode, name string, exp, got Type) {
}
