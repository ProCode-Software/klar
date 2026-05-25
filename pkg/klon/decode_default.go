package klon

import (
	"fmt"
	"reflect"
	"sync"

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
		return d.makeInterfaceDecoder(rt)
	default:
		// Including reflect.Function, Complex, Chan, and UnsafePointer
		return func(rv reflect.Value, val ast.Value, d *decoder) error {
			return decodeError(klonerrs.ErrUnsupportedValue, rv, val,
				"Unsupported Go type %s", rv.Type().String(),
			)
		}
	}
}

func decodeString(rv reflect.Value, v ast.Value, d *decoder) error {
	switch v := v.(type) {
	case *ast.String, *ast.Boolean, *ast.Number:
		str, err := ToString(v)
		if err != nil { // Ex: Variable resolution error
			return err
		}
		rv.SetString(str)
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

func makeMapDecoder(rt reflect.Type) decodeFunc {
	var (
		keyType, valType       = rt.Key(), rt.Elem()
		once                   sync.Once
		decodeKey, decodeValue decodeFunc
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(func() {
			decodeKey = d.getDecoder(keyType)
			decodeValue = d.getDecoder(valType)
		})

		obj, ok := val.(*ast.Object)
		if !ok {
			return typeMismatchError(rv, val)
		}

		if rv.IsNil() {
			rv.Set(reflect.MakeMap(rt))
		}

		for _, f := range obj.Fields {
			kv := reflect.New(keyType).Elem()
			if err := decodeKey(kv, f.Key, d); err != nil {
				return err
			}
			vv := reflect.New(valType).Elem()
			if err := decodeValue(vv, f.Value, d); err != nil {
				return err
			}
			rv.SetMapIndex(kv, vv)
		}
		return nil
	}
}

func (d *decoder) makeStructDecoder(rt reflect.Type) decodeFunc {
	strFields, _ := makeStructFields(rt, d.flags)
	var once sync.Once
	initDecoder := func() {
		for _, field := range strFields.Flat {
			if field.Decode == nil {
				field.Decode = d.getDecoder(field.Type)
			}
		}
	}
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(initDecoder)
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
}

func (d *decoder) decodeField(rv reflect.Value, strFields structFields, f *ast.Field) error {
	if f.Arrow != nil {
		// The object should have already been preprocessed, and that involves resolving ArrowRefs.
		panic("field should not have Arrow during decoding")
	}
	if f.KeyPath != nil {
		return nil // TODO
	}

	// Keys can be strings, bools, or numbers
	name, err := ToString(f.Key)
	if err != nil {
		if err, ok := err.(*Error); ok && err.Code == klonerrs.ErrCantConvertToString {
			// Key type should already be validated at parse-time
			panic(fmt.Sprintf("object key should have been validated during parse: %T", f.Key))
		}
		return err
	}

	field, err := strFields.Lookup(name, f.Key, d.flags)
	if err != nil {
		return err
	}
	// Follow the indices to find the actual field [reflect.Value]. We
	// can't use rv.FieldByIndex because it will panic on a nil pointer.
	fv := rv
	for _, i := range field.Indices {
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				el := fv.Type().Elem()
				// Embedded pointer to unexported struct: type A struct { *b `klon:"B"` }
				if !fv.CanSet() {
					return fmt.Errorf("can't set embedded pointer to unexported struct %s", el)
				}
				fv.Set(reflect.New(el))
			}
			fv = fv.Elem()
		}
		fv = fv.Field(i)
	}

	return field.Decode(fv, f.Value, d)
}

func (d *decoder) makeSliceDecoder(rt reflect.Type) decodeFunc {
	var (
		itemType    = rt.Elem()
		decodeItem  decodeFunc
		once        sync.Once
		initDecoder = func() { decodeItem = d.getDecoder(itemType) }
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(initDecoder)
		list, ok := val.(*ast.List)
		if !ok {
			if d.flags.Has(klonflags.NoSingleItemToArray) {
				return typeMismatchError(rv, val)
			}
			// Create a new slice with a single item
			rv.Set(reflect.MakeSlice(rv.Type(), 1, 1))
			return decodeItem(rv.Index(0), val, d)
		}

		if rv.IsNil() {
			rv.Set(reflect.MakeSlice(rv.Type(), 0, len(list.Items)))
		} else {
			rv.SetLen(0)
		}

		for _, item := range list.Items {
			rest, ok := item.(*ast.ArrowRef)
			if !ok {
				// Normal item
				i := rv.Len()
				rv.Grow(1)
				rv.SetLen(i + 1)
				if err := decodeItem(rv.Index(i), item, d); err != nil {
					return err
				}
				continue
			}

			// Rest
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
			// Append items from the resolved list
			for _, item := range restList.Items {
				i := rv.Len()
				rv.Grow(1)
				rv.SetLen(i + 1)
				if err := decodeItem(rv.Index(i), item, d); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (d *decoder) makeArrayDecoder(rt reflect.Type) decodeFunc {
	var (
		arrLength   = rt.Len()
		itemType    = rt.Elem()
		decodeItem  decodeFunc
		once        sync.Once
		initDecoder = func() { decodeItem = d.getDecoder(itemType) }
	)
	// TODO: allow strings to be used as [...]byte/rune
	return func(rv reflect.Value, val ast.Value, d *decoder) (err error) {
		once.Do(initDecoder)

		list, ok := val.(*ast.List)
		if !ok {
			if d.flags.Has(klonflags.NoSingleItemToArray) {
				return typeMismatchError(rv, val)
			}
			if !d.flags.Has(klonflags.IgnoreArrayLength) && arrLength != 1 {
				if arrLength == 0 {
					return decodeError(klonerrs.ErrWrongArrayLength, rv, val,
						"Expected no items in the list",
					)
				}
				return decodeError(klonerrs.ErrWrongArrayLength, rv, val,
					"Not enough items in the list: Expected %d, but found 1", arrLength,
				)
			}
			if arrLength == 0 {
				return nil
			}
			// Put a single item into an array
			return decodeItem(rv.Index(0), val, d)
		}

		var i int
		for _, item := range list.Items {
			if rest, ok := item.(*ast.ArrowRef); ok {
				// Rest
				if i, err = d.appendRestToArray(rv, list, rest, decodeItem, i, arrLength); err != nil {
					return err
				} else if i >= arrLength {
					break
				}
				continue
			}

			if i >= arrLength {
				// Too many items
				if d.flags.Has(klonflags.IgnoreArrayLength) {
					break
				}
				// Since we don't know how many items we have after this, taking
				// rests in account, we just use the current index and more.
				var plus rune
				if i > arrLength {
					plus = '+'
				}
				return decodeError(klonerrs.ErrWrongArrayLength, rv, list,
					"Too many items in the list: Expected %d, but found %d%c",
					arrLength, i+1, plus,
				)
			}
			if err := decodeItem(rv.Index(i), item, d); err != nil {
				return err
			}
			i++
		}

		// Check if there weren't enough items. (Here 'i' is 1-based)
		if i < arrLength {
			if arrLength == 0 {
				return decodeError(klonerrs.ErrWrongArrayLength, rv, val,
					"Expected no items in the list",
				)
			}
			return decodeError(klonerrs.ErrWrongArrayLength, rv, list,
				"Not enough items in the list: Expected %d, but found %d", arrLength, i,
			)
		}
		return nil
	}
}

func (d *decoder) appendRestToArray(rv reflect.Value, list *ast.List, rest *ast.ArrowRef,
	decodeItem decodeFunc, i, arrLength int,
) (int, error) {
	res, err := d.resolveVar(rest.Var)
	if err != nil {
		return i, err
	}
	restList, ok := res.(*ast.List)
	if !ok {
		if _, ok := res.(*ast.None); ok {
			return i, nil // Rest can be 'none'
		}
		return i, decodeError(klonerrs.ErrInvalidRest, rv, res,
			"'%s' must be a list in order to use it as a rest", rest.Var.Name,
		)
	}

	// Append items from the resolved list
	for _, item := range restList.Items {
		if i >= arrLength {
			if d.flags.Has(klonflags.IgnoreArrayLength) {
				return arrLength, nil
			}
			var plus rune
			if i > arrLength {
				plus = '+'
			}
			return i, decodeError(klonerrs.ErrWrongArrayLength, rv, list,
				"Too many items in the list: Expected %d, but found %d%c",
				arrLength, i+len(restList.Items), plus,
			)
		}
		if err := decodeItem(rv.Index(i), item, d); err != nil {
			return i, err
		}
		i++
	}
	return i, nil
}

func (d *decoder) makePointerDecoder(rt reflect.Type) decodeFunc {
	var (
		elm         = rt.Elem()
		decode      decodeFunc
		once        sync.Once
		initDecoder = func() { decode = d.getDecoder(elm) }
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(initDecoder)
		if rv.IsNil() {
			rv.Set(reflect.New(elm))
		}
		return decode(rv.Elem(), val, d)
	}
}

func (d *decoder) makeInterfaceDecoder(rt reflect.Type) decodeFunc {
	// TODO
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		var v any
		switch node := val.(type) {
		case *ast.String:
			v = node.Raw
		case *ast.Boolean:
			v = node.Value
		case *ast.Number:
			v = node.Value
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
			return decodeError(klonerrs.ErrUnsupportedValue, rv, val,
				"Can't decode %T into any", val,
			)
		}
		if v != nil {
			rv.Set(reflect.ValueOf(v))
		} else {
			rv.SetZero()
		}
		return nil
	}
}
