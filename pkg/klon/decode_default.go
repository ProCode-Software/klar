package klon

import (
	"encoding"
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
		return makeMapDecoder(rt, nil, nil)
	case reflect.Struct:
		return d.makeStructDecoder(rt)
	case reflect.Slice:
		return makeSliceDecoder(rt, nil)
	case reflect.Array:
		return makeArrayDecoder(rt, nil)
	case reflect.Pointer:
		return makePointerDecoder(rt)
	case reflect.Interface:
		return decodeInterface
	default:
		// Including reflect.Function, Complex, Chan, and UnsafePointer
		return func(rv reflect.Value, val ast.Value, d *decoder) error {
			return fmt.Errorf("unsupported Go type: %s", rv.Type().String())
		}
	}
}

func decodeUnmarshaller(rv reflect.Value, val ast.Value, d *decoder) error {
	if !rv.CanAddr() {
		return fmt.Errorf("klon: can't decode into non-addressable value of type %s"+
			"(is the receiver of UnmarshallKlon a pointer?)", rv.Type(),
		)
	}
	rec := rv.Addr().Interface().(Unmarshaller)
	if err := rec.UnmarshallKlon(val); err != nil {
		if _, ok := err.(*Error); ok {
			return err
		}
		return decodeError(klonerrs.ErrUnmarshallerError, rv, val, "%s", err.Error())
	}
	return nil
}

func decodeTextUnmarshaller(rv reflect.Value, val ast.Value, d *decoder) error {
	str, err := d.valueToString(val)
	if err != nil {
		return err
	}
	if !rv.CanAddr() {
		return fmt.Errorf("klon: can't decode into non-addressable value of type %s"+
			"(is the receiver of UnmarshalText a pointer?)", rv.Type(),
		)
	}
	rec := rv.Addr().Interface().(encoding.TextUnmarshaler)
	if err = rec.UnmarshalText([]byte(str)); err != nil {
		return decodeError(klonerrs.ErrUnmarshallerError, rv, val, "%s", err.Error())
	}
	return nil
}

func decodeString(rv reflect.Value, v ast.Value, d *decoder) error {
	switch v := v.(type) {
	case *ast.String, *ast.Boolean, *ast.Number:
		str, err := d.valueToString(v)
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
	if float64(asInt) != num.Value && !d.flags.Has(klonflags.ClampNumbers) {
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
	clamp := d.flags.Has(klonflags.ClampNumbers)
	asUInt := uint64(num.Value)
	if float64(asUInt) != num.Value && !clamp {
		// Truncated
		return decodeError(klonerrs.ErrTruncatedNumber, rv, num,
			"Number '%s' must be a whole integer", num.Source,
		)
	}
	if num.Value < 0 {
		if clamp {
			rv.SetUint(0)
			return nil
		}
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

func makeMapDecoder(rt reflect.Type, decodeKey, decodeValue decodeFunc) decodeFunc {
	var (
		keyType, valType = rt.Key(), rt.Elem()
		once             sync.Once
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(func() {
			if decodeKey == nil {
				decodeKey = d.getDecoder(keyType)
			}
			if decodeValue == nil {
				decodeValue = d.getDecoder(valType)
			}
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
	strFields, err := makeStructFields(rt, d.flags)
	var once sync.Once
	initDecoder := func() {
		for _, field := range strFields.Flat {
			if field.Decode == nil {
				field.Decode = d.getDecoder(field.Type)
			}
		}
	}
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		if err != nil {
			return err
		}
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
	name, err := d.valueToString(f.Key)
	if err != nil {
		if err, ok := err.(*Error); ok && err.Code == klonerrs.ErrCantConvertToString {
			// Key type should already be validated at parse-time
			panic(fmt.Sprintf("object key should have been validated during parse: %T", f.Key))
		}
		return err
	}

	field, err := strFields.Lookup(name, f.Key, d.flags)
	if err != nil {
		if d.flags.Has(klonflags.NoUnknownFields) {
			if d.shouldWarn(err) {
				d.warn(err)
				return nil
			}
			return err
		}
		return nil
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

func makeSliceDecoder(rt reflect.Type, decodeItem decodeFunc) decodeFunc {
	var (
		itemType = rt.Elem()
		once     sync.Once
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(func() {
			if decodeItem == nil {
				decodeItem = d.getDecoder(itemType)
			}
		})
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
			restList, empty, err := resolveRest[*ast.List](d, rest, rv)
			if err != nil {
				return err
			} else if empty {
				continue
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

func makeArrayDecoder(rt reflect.Type, decodeItem decodeFunc) decodeFunc {
	var (
		arrLength = rt.Len()
		itemType  = rt.Elem()
		once      sync.Once
	)
	// TODO: allow strings to be used as [...]byte/rune
	return func(rv reflect.Value, val ast.Value, d *decoder) (err error) {
		once.Do(func() {
			if decodeItem == nil {
				decodeItem = d.getDecoder(itemType)
			}
		})

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
	restList, empty, err := resolveRest[*ast.List](d, rest, rv)
	if err != nil {
		return i, err
	} else if empty {
		return i, nil
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

func makePointerDecoder(rt reflect.Type) decodeFunc {
	var (
		elm    = rt.Elem()
		once   sync.Once
		decode decodeFunc
	)
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		once.Do(func() { decode = d.getDecoder(elm) })
		if rv.IsNil() {
			rv.Set(reflect.New(elm))
		}
		return decode(rv.Elem(), val, d)
	}
}

func decodeInterface(rv reflect.Value, val ast.Value, d *decoder) error {
	// If the interface is set to a pointer, decode into the pointer's value
	if !rv.IsNil() {
		next := rv.Elem() // Underlying value of interface
		if next.Kind() == reflect.Pointer && rv != next.Elem() {
			if next.IsNil() {
				// Initialize the pointer if it's nil
				next = reflect.New(next.Type().Elem())
				rv.Set(next)
			}
			// Attempt to decode into the pointed-to value
			decode := d.getDecoder(next.Type().Elem())
			if err := decode(next.Elem(), val, d); err == nil {
				return nil
			}
			// If the decode fails, fall back to decoding into any.
		}
	}
	// Decode into 'any'
	// We can't decode into a nil interface with methods
	if rv.NumMethod() != 0 {
		return fmt.Errorf("can't decode into nil interface with methods: %s", rv.Type().String())
	}
	v, err := d.toGoValue(val)
	switch {
	case err != nil:
		return err
	case v != nil:
		rv.Set(reflect.ValueOf(v))
	default:
		rv.SetZero()
	}
	return nil
}
