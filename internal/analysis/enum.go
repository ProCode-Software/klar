package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) parseEnum(t ast.EnumDeclaration) types.Enum {
	type pendingItem struct {
		name string
		pos  ranges.Range
	}
	var (
		members = make(map[string]any, len(t.Values))
		last    any
		expType Type
		pending = make(map[string]pendingItem)
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
			if expType == types.Float {
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
			pending[i.Identifier] = pendingItem{v.Identifier, v.Range}
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
	for k, item := range pending {
		if v, ok := members[item.name]; ok {
			members[k] = v
		} else {
			c.Error(errors.Range(errors.ErrEnumUndefined, item.pos))
		}
	}
	return types.Enum{
		ValueType: expType,
		Members:   members,
	}
}
