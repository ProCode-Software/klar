package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) parseEnum(t ast.EnumDeclaration) types.Enum {
	var (
		members = make(map[string]any, len(t.Values))
		last    any
		expType Type
	)
	for _, i := range t.Values {
		if _, ok := members[i.Identifier]; ok {
			c.Error(errors.Range(errors.ErrRedeclaredEnum, i.Range))
			continue
		}
		var currType Type
		switch v := i.Value.(type) {
		// No value
		case nil:
			switch last := last.(type) {
			case string:
				// Infer string enum as their name
				last = i.Identifier
			case int64:
				last += 1
			case float64:
				last += 1
			case nil:
				// First enum member, int by default
				last, currType = 0, types.Int
			}
		case ast.IntegerLiteral:
			// Allow ints for floats
			if _, ok := last.(float64); ok {
				last, currType = float64(v.Value), types.Float
			} else {
				last, currType = v.Value, types.Int
			}
		case ast.FloatLiteral:
			last, currType = v.Value, types.Float
		case ast.StringLiteral:
			last, currType = v.Content, types.String
		case ast.Symbol:
			// Wait for it to be assigned
		default:
			c.Error(errors.Range(errors.ErrInvalidEnumValue, i.Range))
		}
		if expType == nil {
			expType = currType
		} else if currType != expType {
			c.Error(errors.TypeMismatch(expType, currType, i.Range))
		}
		members[i.Identifier] = last
	}
	return types.Enum{
		ValueType: expType,
		Members:   members,
	}
}
