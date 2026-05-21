package klon

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

func (d *decoder) makeDefaultDecoder(rt reflect.Type) decodeFunc {
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
		return d.makeSliceDecoder(rt)
	case reflect.Array:
		return d.makeArrayDecoder(rt)
	case reflect.Pointer:
		return makePointerDecoder(rt)
	case reflect.Interface:
		return makeInterfaceDecoder(rt)
	default:
		return decodeInvalid
	}
}

func decodeString(rv reflect.Value, v ast.Value, d *decoder) error {
	switch v := v.(type) {
	case *ast.String:
		rv.SetString(v.Raw) // TODO
	case *ast.Boolean:
		rv.SetString(strconv.FormatBool(v.Value))
	case *ast.Number:
		rv.SetString(v.Source)
	case *ast.None:
	default:
		return typeError(rv, v)
	}
	return nil
}

func decodeBool(rv reflect.Value, val ast.Value, d *decoder) error {
	switch val := val.(type) {
	case *ast.Boolean:
		rv.SetBool(val.Value)
	case *ast.None:
	default:
		return typeError(rv, val)
	}
	return nil
}

func decodeInt(rv reflect.Value, val ast.Value, d *decoder) error {
	switch val := val.(type) {
	case *ast.Number:
		asInt := int64(val.Value)
		if float64(asInt) != val.Value {
			// Truncated
			return decodeError(ErrTruncatedNumber, rv, val,
				"Number %f must be a whole integer to be stored in Go type %s",
				val.Value, rv.Type().String(),
			)
		}
		rv.SetInt(asInt)
	case *ast.None:
	default:
		return typeError(rv, val)
	}
	return nil
}

func decodeUInt(rv reflect.Value, val ast.Value, d *decoder) error {
	switch val := val.(type) {
	case *ast.Number:
		if val.Value < 0 {
			return decodeError(ErrNegativeNumber, rv, val,
				"Can't decode negative number %f into Go type %s",
				val.Value, rv.Type().String(),
			)
		}
		rv.SetUint(uint64(val.Value))
		return nil
	case *ast.None:
	}
	return typeError(rv, val)
}

func decodeFloat(rv reflect.Value, val ast.Value, d *decoder) error {
	switch val := val.(type) {
	case *ast.Number:
		rv.SetFloat(val.Value)
	case *ast.None:
	default:
		return typeError(rv, val)
	}
	return nil
}

// Including reflect.Function, Chan, and UnsafePointer
func decodeInvalid(rv reflect.Value, val ast.Value, d *decoder) error {
	return decodeError(ErrUnsupportedValue, rv, val,
		"Unsupported Go type %s", rv.Type().String(),
	)
}

// TODO
func makeInterfaceDecoder(rt reflect.Type) decodeFunc {
	return nil
}

func (d *decoder) makeStructDecoder(rt reflect.Type) decodeFunc {
	strFields, _ := makeStructFields(rt, d.flags)

	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		switch val := val.(type) {
		case *ast.Object:
			for _, f := range val.Fields {
				if err := d.decodeField(rv, strFields, f); err != nil {
					return err
				}
			}
		case *ast.List:
			for _, item := range val.Items {
				if obj, ok := item.(*ast.Object); ok {
					for _, f := range obj.Fields {
						if err := d.decodeField(rv, strFields, f); err != nil {
							return err
						}
					}
				} else {
					// Single field in dashed list
					// e.g. - key: value
					// Wait, parseObject returns *ast.Object if it has colons.
					// If it's a dashed list, it might contain objects.
				}
			}
		default:
			return typeError(rv, val)
		}
		return nil
	}
}

func (d *decoder) decodeField(rv reflect.Value, strFields StructFields, f *ast.Field) error {
	if f.Key == nil {
		if f.Value != nil {
			// This might be an ArrowRef (spread)
			if arrow, ok := f.Value.(*ast.ArrowRef); ok {
				return d.decodeValue(arrow, rv)
			}
		}
		return nil
	}
	keyStr, ok := f.Key.(*ast.String)
	if !ok {
		return nil
	}
	lower := strings.ToLower(keyStr.Raw)
	if field, ok := strFields.Fields[lower]; ok {
		fieldVal := rv.FieldByIndex(field.Indices)
		return d.decodeValue(f.Value, fieldVal)
	}
	return nil
}

func (d *decoder) makeSliceDecoder(rt reflect.Type) decodeFunc {
	return nil
}

func (d *decoder) makeArrayDecoder(rt reflect.Type) decodeFunc {
	arrLength := rt.Len()
	itemType := rt.Elem()
	decodeItem := d.getDecoder(itemType)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		switch val := val.(type) {
		case *ast.List:
			if !d.flags.Has(IgnoreArrayLength) && len(val.Items) != arrLength {
				return decodeError(ErrWrongArrayLength, rv, val,
					"Expected %d items, but found %d", arrLength, len(val.Items),
				)
			}
			for i := range min(len(val.Items), arrLength) {
				if err := decodeItem(rv.Index(i), val.Items[i], d); err != nil {
					return err
				}
			}
		case *ast.None:
		default:
			if d.flags.Has(NoSingleItemToArray) {
				return typeError(rv, val)
			}
			if !d.flags.Has(IgnoreArrayLength) && arrLength != 1 {
				return decodeError(ErrWrongArrayLength, rv, val,
					"Expected %d items, but found 1", arrLength,
				)
			}
			return decodeItem(rv.Index(0), val, d)
		}
		return nil
	}
}
