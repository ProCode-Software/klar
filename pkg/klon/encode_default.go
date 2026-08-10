package klon

import (
	"bytes"
	"cmp"
	"fmt"
	"reflect"
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
	case reflect.Slice:
		return makeSliceEncoder(rt, nil)
	case reflect.Array:
		return makeArrayEncoder(rt, nil)
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
	return nil
}

func encodeTextMarshaler(rv reflect.Value, e *encoder) error {
	return nil
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

func makeMapEncoder(rt reflect.Type, _, _ any) encodeFunc {
	return nil
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
		once.Do(initFields)
		if len(fields.Flat) == 0 {
			// Empty object literal for structs with no fields
			return e.write("{}")
		}

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
				if err := cmp.Or(e.writeByte('\n'), e.appendIndent()); err != nil {
					return err
				}
			} else if printInline && i > 0 {
				if err := e.write(", "); err != nil {
					return err
				}
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
		return e.write("\n\n") // 2 newlines after last field
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
	return cmp.Or(
		e.writeSlice(escapedName), e.write(": "), f.Encode(fv, e),
	)
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
		panic("appendIndent called with indentSize == 0")
	}
	useTabs := e.flags.Has(klonflags.IndentWithTabs)
	var indentChar byte = ' '
	if useTabs {
		indentChar = '\t'
	}
	spaces := repeatChar(e.indentSize*e.indentLevel, indentChar)
	dashes := repeatChar(e.indentLevel, '-')
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

func makeSliceEncoder(rt reflect.Type, _ any) encodeFunc {
	return nil
}

func makeArrayEncoder(rt reflect.Type, _ any) encodeFunc {
	return nil
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
