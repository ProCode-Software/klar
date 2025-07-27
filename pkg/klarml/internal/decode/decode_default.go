package decode

import (
	"reflect"
	"strconv"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
)

func (d *Decoder) makeDefaultDecoder(rt reflect.Type) decodeFunc {
	kind := rt.Kind()
	switch kind {
	case reflect.String:
		return decodeString
	case reflect.Bool:
		return decodeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return decodeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return decodeUInt
	case reflect.Float32, reflect.Float64:
		return decodeFloat
	case reflect.Map:
		return makeMapDecoder(rt)
	case reflect.Struct:
		return d.makeStructDecoder(rt)
	case reflect.Slice:
		return makeSliceDecoder(rt)
	case reflect.Array:
		return makeArrayDecoder(rt)
	case reflect.Pointer:
		return makePointerDecoder(rt)
	case reflect.Interface:
		return makeInterfaceDecoder(rt)
	default:
		return decodeInvalid
	}
}

func decodeString(rv reflect.Value, d *Decoder) (ast.Node, error) {
	v, err := d.ReadValue()
	if err != nil {
		return v, err
	}
	switch v := v.(type) {
	case *ast.String:
		rv.SetString(v.Value)
	case *ast.Bool:
		rv.SetString(strconv.FormatBool(v.Value))
	case *ast.Number:
		rv.SetString(v.Source)
	case *ast.Null:
	default:
		return v, d.TypeError(rv, v)
	}
	return v, nil
}

func decodeBool(rv reflect.Value, d *Decoder) (ast.Node, error) {
	val, err := d.ReadValue()
	if err != nil {
		return val, err
	}
	switch val := val.(type) {
	case *ast.Bool:
		rv.SetBool(val.Value)
	case *ast.Null:
	default:
		return val, d.TypeError(rv, val)
	}
	return val, nil
}

func decodeInt(rv reflect.Value, d *Decoder) (ast.Node, error) {
	val, err := d.ReadValue()
	if err != nil {
		return val, err
	}
	switch val := val.(type) {
	case *ast.Number:
		asInt := int64(val.Value)
		if float64(asInt) != val.Value {
			// Truncated
			return val, &errors.NumberRangeError{
				Value:     val.Value,
				Truncated: true,
				Expected:  rv.Type(),
			}
		}
		rv.SetInt(asInt)
	case *ast.Null:
	default:
		return val, d.TypeError(rv, val)
	}
	return val, nil
}

func decodeUInt(rv reflect.Value, d *Decoder) (ast.Node, error) {
	val, err := d.ReadValue()
	if err != nil {
		return val, err
	}
	switch val := val.(type) {
	case *ast.Number:
		if val.Value < 0 {
			return val, &errors.NumberRangeError{
				Value:     val.Value,
				Truncated: false,
				Expected:  rv.Type(),
			}
		}
		rv.SetUint(uint64(val.Value))
		return val, nil
	case *ast.Null:
	}
	return val, d.TypeError(rv, val)
}

func decodeFloat(rv reflect.Value, d *Decoder) (ast.Node, error) {
	val, err := d.ReadValue()
	if err != nil {
		return val, err
	}
	switch val := val.(type) {
	case *ast.Number:
		rv.SetFloat(val.Value)
	case *ast.Null:
	default:
		return val, d.TypeError(rv, val)
	}
	return val, nil
}

func decodeInvalid(rv reflect.Value, d *Decoder) (ast.Node, error) {
	return nil, nil
}

func makeInterfaceDecoder(rt reflect.Type) decodeFunc {
	return nil
}
