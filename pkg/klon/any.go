package klon

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
)

func (d *decoder) decodeAny(rv reflect.Value, val ast.Value) error {
	// TODO: use the klonflags
	var v any
	switch node := val.(type) {
	case *ast.String:
		v = node.Raw // TODO
	case *ast.Boolean:
		v = node.Value
	case *ast.Number:
		if node.Float {
			v = node.Value
		} else {
			v = int(node.Value)
		}
	case *ast.List:
		l := make([]any, 0, len(node.Items))
		for _, item := range node.Items {
			if rest, ok := item.(*ast.ArrowRef); ok {
				// Resolve and preprocess the rest list
				res, err := d.resolveVar(rest.Var)
				if err != nil {
					return err
				}
				restList, ok := res.(*ast.List)
				if !ok {
					if _, ok := res.(*ast.None); ok {
						continue // Rest can be 'none'
					}
					return decodeError(klonerrs.ErrInvalidRest, rv, res,
						"'%s' must be a list in order to use it as a rest", rest.Var.Name,
					)
				}
				// Recursively decode items from the resolved list
				for _, subItem := range restList.Items {
					var itemAny any
					if err := d.decodeValue(subItem, reflect.ValueOf(&itemAny).Elem()); err != nil {
						return err
					}
					l = append(l, itemAny)
				}
				continue
			}

			var itemAny any
			if err := d.decodeValue(item, reflect.ValueOf(&itemAny).Elem()); err != nil {
				return err
			}
			l = append(l, itemAny)
		}
		v = l
	case *ast.Object:
		m := make(map[string]any)
		for _, f := range node.Fields {
			name, err := ToString(f.Key)
			if err != nil {
				return err
			}
			var valAny any
			if err := d.decodeValue(f.Value, reflect.ValueOf(&valAny).Elem()); err != nil {
				return err
			}
			m[name] = valAny
		}
		v = m
	case *ast.None:
		v = nil
	default:
		panic(fmt.Sprintf("unhandled klon node while decoding into any: %T", val))
	}
	if v != nil {
		rv.Set(reflect.ValueOf(v))
	} else {
		rv.SetZero()
	}
	return nil
}
