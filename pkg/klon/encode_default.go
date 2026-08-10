package klon

import (
	"bytes"
	"cmp"
	"encoding"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"sync"

	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func (e *encoder) makeDefaultEncoder(rt reflect.Type) encodeFunc {
	switch rt.Kind() {
	case reflect.String:
		return encodeString
	case reflect.Bool:
		return encodeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return encodeUInt
	case reflect.Float32, reflect.Float64:
		return encodeFloat
	case reflect.Map:
		return makeMapEncoder(rt, nil, nil)
	case reflect.Struct:
		return e.makeStructEncoder(rt)
	case reflect.Slice, reflect.Array:
		return makeSliceOrArrayEncoder(rt, nil)
	case reflect.Pointer:
		return makePointerEncoder(rt)
	case reflect.Interface:
		return encodeInterface
	default:
		// Including reflect.Function, Complex, Chan, and UnsafePointer
		return func(rv reflect.Value, e *encoder) error {
			return fmt.Errorf("unsupported Go type: %s", rv.Type().String())
		}
	}
}

func encodeMarshaller(rv reflect.Value, e *encoder) error {
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return e.write("none")
	}
	marshaller, ok := reflect.TypeAssert[Marshaller](rv)
	if !ok && rv.CanAddr() {
		// Try on a pointer
		marshaller, _ = reflect.TypeAssert[Marshaller](rv.Addr())
	}
	if marshaller == nil {
		// I actually don't know how we get here
		return e.write("none")
	}
	res, err := marshaller.MarshallKlon()
	if err != nil {
		return fmt.Errorf(
			"%s.MarshallKlon() returned an error: %w", rv.Type().String(), err,
		)
	}
	return e.writeSlice(res)
}

func encodeTextMarshaler(rv reflect.Value, e *encoder) error {
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return e.write("none")
	}
	tm, ok := reflect.TypeAssert[encoding.TextMarshaler](rv)
	if !ok && rv.CanAddr() {
		// Try on a pointer
		tm, _ = reflect.TypeAssert[encoding.TextMarshaler](rv.Addr())
	}
	if tm == nil {
		// I actually don't know how we get here
		return e.write("none")
	}
	res, err := tm.MarshalText()
	if err != nil {
		return fmt.Errorf(
			"%s.MarshalText() returned an error: %w", rv.Type().String(), err,
		)
	}
	// Since it is *Text*, its output is considered a string and needs to be quoted
	return e.writeSlice(Quote(res))
}

func encodeString(rv reflect.Value, e *encoder) error {
	str := Quote([]byte(rv.String()))
	return e.writeSlice(str)
}

func encodeBool(rv reflect.Value, e *encoder) error {
	if rv.Bool() == true {
		return e.write("true")
	}
	return e.write("false")
}

func encodeInt(rv reflect.Value, e *encoder) error {
	return e.appendInt(rv.Int())
}

func (e *encoder) appendInt(num int64) error {
	if e.buf != nil {
		e.buf = strconv.AppendInt(e.buf, num, 10)
		return nil
	}
	_, err := e.writer.Write(strconv.AppendInt(nil, num, 10))
	return err
}

func encodeUInt(rv reflect.Value, e *encoder) error {
	return e.appendUInt(rv.Uint())
}

func (e *encoder) appendUInt(num uint64) error {
	base := 10
	if e.flags.Has(klonflags.HexadecimalUInt) {
		base = 16
		if err := e.write("0x"); err != nil {
			return err
		}
	}
	if e.buf != nil {
		e.buf = strconv.AppendUint(e.buf, num, base)
		return nil
	}
	_, err := e.writer.Write(strconv.AppendUint(nil, num, base))
	return err
}

func encodeFloat(rv reflect.Value, e *encoder) error {
	return e.appendFloat(rv.Float(), rv.Type().Bits())
}

func (e *encoder) appendFloat(num float64, bitSize int) error {
	if e.buf != nil {
		e.buf = strconv.AppendFloat(e.buf, num, 'f', -1, bitSize)
		return nil
	}
	_, err := e.writer.Write(strconv.AppendFloat(nil, num, 'f', -1, bitSize))
	return err
}

func makeMapEncoder(rt reflect.Type, encodeKey, encodeValue encodeFunc) encodeFunc {
	key, val := rt.Key(), rt.Elem()
	var once sync.Once
	return func(rv reflect.Value, e *encoder) (err error) {
		// This logic is similar to [encoder.makeStructEncoder]
		if rv.Len() == 0 {
			return e.write("{}") // Including nil maps
		}
		once.Do(func() {
			if encodeKey == nil {
				encodeKey = e.getEncoder(key)
				// TODO: Validate Go key type
				// TODO: Should strings with spaces be quoted specially?
			}
			if encodeValue == nil {
				encodeValue = e.getEncoder(val)
			}
		})

		printInline := e.indentSize <= 0
		if printInline {
			if err := e.write("{ "); err != nil {
				return err
			}
		}
		e.indentLevel++

		// Sort the keys in the map
		keys := rv.MapKeys()
		if !e.flags.Has(klonflags.NoSortMaps) {
			slices.SortFunc(keys, compare)
		}
		// I have a concern over keys with the same Klon representation. If we
		// don't allow interfaces as map keys, this won't be a problem. Otherwise
		// if we check the concrete type of each key, we have to keep a map of
		// seen keys in Klon form.
		for i, key := range keys {
			val := rv.MapIndex(key)
			if !val.IsValid() {
				panic(fmt.Sprintf(
					"value for map key %s obtained from reflect.Value.MapKeys() invalid",
					key.String(),
				))
			}
			// Newline or comma between entries
			if !printInline && (i > 0 || e.indentLevel > 1) {
				// Also write a newline before the first field, except at the top-level
				err = cmp.Or(e.writeByte('\n'), e.appendIndent())
			} else if printInline && i > 0 {
				err = e.write(", ")
			}
			if err != nil {
				return err
			}
			// TODO: If the field's value was nil, separators would still be printed. Fix
			colon := ":"
			if !e.isPrintedMultiline(val) {
				colon += " "
			}
			if err := cmp.Or(
				encodeKey(key, e), e.write(colon), encodeValue(val, e),
			); err != nil {
				return err
			}
		}
		e.indentLevel--
		if printInline {
			return e.write(" }")
		}
		return e.writeByte('\n') // After the last field
	}
}

func (e *encoder) makeStructEncoder(rt reflect.Type) encodeFunc {
	fields, err := makeStructFields(rt, e.flags)
	initFields := func() {
		for _, field := range fields.Flat {
			if field.Encode == nil {
				field.Encode = e.getEncoder(field.Type)
			}
		}
	}
	var once sync.Once
	return func(rv reflect.Value, e *encoder) error {
		if err != nil {
			return err // Error while creating struct fields
		}
		if len(fields.Flat) == 0 {
			// Empty object literal for structs with no fields
			return e.write("{}")
		}
		once.Do(initFields)

		printInline := e.indentSize <= 0
		if printInline {
			if err := e.write("{ "); err != nil {
				return err
			}
		}
		e.indentLevel++
		// TODO: Check if fields are being duplicated if [klonflags.KeyedEmbeddedFields] is on
		for i, field := range fields.Flat {
			// Newline or comma between entries
			if !printInline && (i > 0 || e.indentLevel > 1) {
				// Also write a newline before the first field, except at the top-level
				err = cmp.Or(e.writeByte('\n'), e.appendIndent())
			} else if printInline && i > 0 {
				err = e.write(", ")
			}
			if err != nil {
				return err
			}
			// TODO: If the field's value was nil, separators would still be printed. Fix
			if err := e.encodeField(rv, field); err != nil {
				return err
			}
		}
		e.indentLevel--
		if printInline {
			return e.write(" }")
		}
		return e.writeByte('\n') // After the last field
	}
}

func (e *encoder) encodeField(structVal reflect.Value, f *structField) error {
	// Follow the path to reach the value
	fv := structVal
	if len(f.Indices) == 0 {
		fv = structVal.Field(f.Indices[0])
	} else {
		for _, i := range f.Indices {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return nil
				}
				fv = fv.Elem()
			}
			fv = fv.Field(i)
		}
	}
	// This package omits empty/zero fields by default, unlike JSON
	if !e.flags.Has(klonflags.PreserveEmptyFields) && isEmpty(fv) {
		return nil
	}

	escapedName := Quote([]byte(f.Name))
	// If the value is an object, don't print a space after the colon
	// before the newline
	colon := ":"
	if !e.isPrintedMultiline(fv) {
		colon += " "
	}
	return cmp.Or(
		e.writeSlice(escapedName), e.write(colon), f.Encode(fv, e),
	)
}

func (e *encoder) isPrintedMultiline(rv reflect.Value) bool {
	if e.indentSize <= 0 {
		return false
	}
	if rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
		if rv.IsNil() {
			return false
		}
	}
	switch rv.Kind() {
	case reflect.Struct:
		return rv.NumField() > 0
	case reflect.Map:
		return rv.Len() > 0
	}
	return false
}

func isEmpty(rv reflect.Value) bool {
	if rv.IsZero() {
		return true
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	}
	return false
}

func (e *encoder) appendIndent() error {
	if e.indentSize <= 0 {
		// Inline object printing should be handled by the caller
		panic("appendIndent called with indentSize <= 0")
	}
	if e.indentLevel == 1 {
		return nil // Top-level object. No indent needed
	}
	useTabs := e.flags.Has(klonflags.IndentWithTabs)
	var indentChar byte = ' '
	if useTabs {
		indentChar = '\t'
	}
	spaces := repeatChar(e.indentSize*(e.indentLevel-1), indentChar)
	dashes := repeatChar(e.indentLevel-1, '-')
	return cmp.Or(e.writeSlice(spaces), e.writeSlice(dashes), e.writeByte(' '))
}

// Enough for most cases
var (
	defaultSpaces = []byte("                                                  ")
	defaultDashes = []byte("--------------------------------------------------")
)

func repeatChar(n int, c byte) []byte {
	switch {
	case c == ' ' && n <= len(defaultSpaces):
		return defaultSpaces[:n]
	case c == '-' && n <= len(defaultDashes):
		return defaultDashes[:n]
	}
	return bytes.Repeat(defaultSpaces[:1], n)
}

func makeSliceOrArrayEncoder(rt reflect.Type, encodeItem encodeFunc) encodeFunc {
	elem := rt.Elem()
	// If the slice is []byte, write as a string. Note that the element could
	// be a named type derived from byte, so check that it implements
	// neither marshaller interface
	ptr := reflect.PointerTo(rt)
	if elem.Kind() == reflect.Uint8 &&
		!ptr.Implements(marshallerType) && !ptr.Implements(textMarshalerType) {
		return func(rv reflect.Value, e *encoder) error {
			// Sonnet JSON writes bytes as base64 strings, but we'll write as
			// escaped strings, mostly due to Klon's distinct use cases as a
			// user-oriented configuration format.
			return e.writeSlice(Quote(rv.Bytes()))
		}
	}

	var once sync.Once
	return func(rv reflect.Value, e *encoder) (err error) {
		// This logic is similar to [encoder.makeStructEncoder]
		if rv.Len() == 0 {
			return e.write("[]") // Nil slices are written as empty lists, not `none`
		}
		once.Do(func() {
			if encodeItem == nil {
				encodeItem = e.getEncoder(elem)
			}
		})
		// TODO: Detect cycles
		printInline := e.indentSize <= 0
		if printInline {
			if err := e.writeByte('['); err != nil {
				return err
			}
		}
		e.indentLevel++
		for i := range rv.Len() {
			// Newline or comma between items
			if !printInline && (i > 0 || e.indentLevel > 1) {
				// Also write a newline before the first item, except at the top-level
				err = cmp.Or(e.writeByte('\n'), e.appendIndent())
			} else if printInline && i > 0 {
				err = e.write(", ")
			}
			if err != nil {
				return err
			}
			// TODO: The whole list needs to be printed differently if the item
			// is an object (could be in an interface)
			if err := encodeItem(rv.Index(i), e); err != nil {
				return err
			}
		}
		e.indentLevel--
		if printInline {
			return e.writeByte(']')
		}
		return e.writeByte('\n') // After the last item
	}
}

func makePointerEncoder(rt reflect.Type) encodeFunc {
	var (
		elem       = rt.Elem()
		encodeElem encodeFunc
		once       sync.Once
	)
	return func(rv reflect.Value, e *encoder) error {
		if rv.IsNil() {
			return e.write("none")
		}
		once.Do(func() { encodeElem = e.getEncoder(elem) })

		// TODO: Detect cycles
		return encodeElem(rv.Elem(), e)
	}
}

func encodeInterface(rv reflect.Value, e *encoder) error {
	return e.appendAny(rv.Interface())
}

func compare(a, b reflect.Value) int {
	if a.Kind() != b.Kind() {
		panic(fmt.Sprintf(
			"can't compare reflect.Values of different types (%s and %s)",
			a.Kind(), b.Kind(),
		))
	}
	switch a.Kind() {
	case reflect.String:
		return cmp.Compare(a.String(), b.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(a.Uint(), b.Uint())
	case reflect.Float32, reflect.Float64:
		return cmp.Compare(a.Float(), b.Float())
	case reflect.Bool:
		// false < true
		a, b := a.Bool(), b.Bool()
		if a == b {
			return 0
		}
		if a == true {
			return 1
		}
		return -1
	default:
		// Worst case scenario. Shouldn't happen for map keys because all valid
		// Klon object keys were already handled. Don't use [reflect.Value.String]
		// because its result is based on the type, and will most likely be equal.
		return cmp.Compare(fmt.Sprintf("%#v", a), fmt.Sprintf("%#v", b))
	}
}
