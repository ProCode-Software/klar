package klon

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func (d *decoder) toGoValue(val ast.Value) (any, error) {
	switch val := val.(type) {
	case *ast.String:
		str, err := d.evaluateString(val)
		switch {
		case err != nil:
			return nil, err
		case d.flags.Has(klonflags.UseByteSlice):
			return []byte(str), nil
		case d.flags.Has(klonflags.UseRuneSlice):
			return []rune(str), nil
		}
		return str, nil
	case *ast.Boolean:
		if d.flags.Has(klonflags.BoolIsString) {
			return val.String(), nil
		}
		return val.Value, nil
	case *ast.Number:
		switch {
		case d.flags.Has(klonflags.NumberIsString):
			if val.Float {
				return strconv.FormatFloat(val.Value, 'f', -1, 64), nil
			}
			return strconv.FormatInt(int64(val.Value), 10), nil
		case val.Float, d.flags.Has(klonflags.UseFloat64):
			return val.Value, nil

		// Flags below only apply to integers
		case d.flags.Has(klonflags.UseInt64):
			return int64(val.Value), nil
		case d.flags.Has(klonflags.UseInt):
			fallthrough
		default:
			return int(val.Value), nil
		}
	case *ast.List:
		return d.listToGoSlice(val)
	case *ast.Object:
		return d.objectToGoMap(val)
	case *ast.None:
		panic("ast.None shouldn't be here")
	default:
		panic(fmt.Sprintf("unhandled klon node while decoding into any: %T", val))
	}
}

func (d *decoder) listToGoSlice(l *ast.List) ([]any, error) {
	list := make([]any, 0, len(l.Items))
	for _, item := range l.Items {
		rest, ok := item.(*ast.ArrowRef)
		if !ok {
			// Normal item
			val, err := d.toGoValue(item)
			if err != nil {
				return nil, err
			}
			list = append(list, val)
			continue
		}

		// Rest
		restList, empty, err := resolveRest[*ast.List](d, rest, reflect.Value{})
		if err != nil {
			return nil, err
		} else if empty {
			continue
		}
		// Convert rest list items to Go values
		for _, item := range restList.Items {
			val, err := d.toGoValue(item)
			if err != nil {
				return nil, err
			}
			list = append(list, val)
		}
	}
	return list, nil
}

func (d *decoder) objectToGoMap(o *ast.Object) (map[string]any, error) {
	m := make(map[string]any, len(o.Fields))
	for _, f := range o.Fields {
		val, err := d.toGoValue(f.Value)
		switch {
		case err != nil:
			return nil, err
		case val == nil && d.flags.Has(klonflags.OmitNullFields):
			continue
		case f.Key != nil:
			// Normal key
			name, err := d.valueToString(f.Key)
			if err != nil {
				return nil, err
			}
			m[name] = val
		default:
			// Key path
			// TODO
		}
	}
	return m, nil
}

func (e *encoder) appendAny(val any) error {
	// Handle common types without reflection
	switch val := val.(type) {
	case nil:
		return e.write("none")
	case string:
		str := Quote([]byte(val))
		return e.writeSlice(str)
	case []byte:
		str := Quote(val)
		return e.writeSlice(str)
	case float64:
		return e.appendFloat(val, 64)
	case bool:
		if val == true {
			return e.write("true")
		}
		return e.write("false")
	case int:
		return e.appendInt(int64(val))
	case int64:
		return e.appendInt(val)
	case []any:
	case map[string]any:
	default:
		rv := reflect.ValueOf(val)
		encode := e.getEncoder(rv.Type())
		return encode(rv, e)
	}
	return nil
}
