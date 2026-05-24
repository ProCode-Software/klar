package klon

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
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
		return d.makePointerDecoder(rt)
	case reflect.Interface:
		return makeInterfaceDecoder(rt)
	default:
		// Including reflect.Function, Chan, and UnsafePointer
		return func(rv reflect.Value, val ast.Value, d *decoder) error {
			return decodeError(klonerrs.ErrUnsupportedValue, rv, val,
				"Unsupported Go type %s", rv.Type().String(),
			)
		}
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
	default:
		return typeMismatchError(rv, v)
	}
	return nil
}

func decodeBool(rv reflect.Value, val ast.Value, d *decoder) error {
	if bool, ok := val.(*ast.Boolean); ok {
		rv.SetBool(bool.Value)
		return nil
	}
	return typeMismatchError(rv, val)
}

func decodeInt(rv reflect.Value, val ast.Value, d *decoder) error {
	num, ok := val.(*ast.Number)
	if !ok {
		return typeMismatchError(rv, val)
	}
	asInt := int64(num.Value)
	if float64(asInt) != num.Value {
		// Truncated
		return decodeError(klonerrs.ErrTruncatedNumber, rv, num,
			"Number '%s' must be a whole integer", num.Source,
		)
	}
	rv.SetInt(asInt)
	return nil
}

func decodeUInt(rv reflect.Value, val ast.Value, d *decoder) error {
	num, ok := val.(*ast.Number)
	if !ok {
		return typeMismatchError(rv, val)
	}
	asUInt := uint64(num.Value)
	if float64(asUInt) != num.Value {
		// Truncated
		return decodeError(klonerrs.ErrTruncatedNumber, rv, num,
			"Number '%s' must be a whole integer", num.Source,
		)
	}
	if num.Value < 0 {
		return decodeError(klonerrs.ErrNegativeNumber, rv, num,
			"Number '%s' can't be negative", num.Source,
		)
	}
	rv.SetUint(asUInt)
	return nil
}

func decodeFloat(rv reflect.Value, val ast.Value, d *decoder) error {
	if num, ok := val.(*ast.Number); ok {
		rv.SetFloat(num.Value)
		return nil
	}
	return typeMismatchError(rv, val)
}

func (d *decoder) makePointerDecoder(rt reflect.Type) decodeFunc {
	elm := rt.Elem()
	decode := d.getDecoder(elm)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		if rv.IsNil() {
			rv.Set(reflect.New(elm))
		}
		return decode(rv.Elem(), val, d)
	}
}

// TODO
func makeInterfaceDecoder(rt reflect.Type) decodeFunc {
	return nil
}

func (d *decoder) makeStructDecoder(rt reflect.Type) decodeFunc {
	strFields, _ := makeStructFields(rt, d.flags)

	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		obj, ok := val.(*ast.Object)
		if !ok {
			return typeMismatchError(rv, val)
		}
		for _, f := range obj.Fields {
			if err := d.decodeField(rv, strFields, f); err != nil {
				return err
			}
		}
		return nil
	}
	reflect.Value{}.gr
}

func (d *decoder) decodeField(rv reflect.Value, strFields StructFields, f *ast.Field) error {
	if f.Arrow != nil {
		// The object should have already been preprocessed, and that involves resolving ArrowRefs.
		panic("field should not have Arrow during decoding")
	}
	keyStr, ok := f.Key.(*ast.String)
	if !ok {
		return nil
	}
	lower := strings.ToLower(keyStr.Raw)
	if field, ok := strFields.Fields[lower]; ok {
		fieldVal := rv.FieldByIndex(field.Indices)
		if field.Decode == nil {
			field.Decode = d.getDecoder(field.Type)
		}
		return field.Decode(fieldVal, f.Value, d)
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
	// TODO: allow strings to be used as [...]byte/rune
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		list, ok := val.(*ast.List)
		ignoreLen := d.flags.Has(klonflags.IgnoreArrayLength)
		switch {
		case ok && !ignoreLen && len(list.Items) != arrLength:
			// Mismatched list length
			return decodeError(klonerrs.ErrWrongArrayLength, rv, list,
				"Expected %d items, but found %d", arrLength, len(list.Items),
			)
		case ok:
			// Valid list
			minLen := min(len(list.Items), arrLength)
			for i := 0; i < minLen; i++ {
				if rest, ok := list.Items[i].(*ast.ArrowRef); ok {
					// TODO: resolve arrow ref. make rv.Index(_) and list.Items[_] will then be out of sync.
					// Also make sure to validate the new length of the Klon list.
					_ = rest
				}
				if err := decodeItem(rv.Index(i), list.Items[i], d); err != nil {
					return err
				}
			}
		case d.flags.Has(klonflags.NoSingleItemToArray):
			return typeMismatchError(rv, val)
		case !ignoreLen && arrLength != 1:
			return decodeError(klonerrs.ErrWrongArrayLength, rv, val,
				"Expected %d items, but found 1", arrLength,
			)
		default:
			// Put a single item into an array
			return decodeItem(rv.Index(0), val, d)
		}
		return nil
	}
}
