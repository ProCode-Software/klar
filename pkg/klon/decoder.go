package klon

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

var (
	unmarshallerType    = reflect.TypeFor[Unmarshaller]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

type decoder struct {
	ctx   *Context
	flags klonflags.Flags
	vars  map[string]ast.Value
}

type decodeFunc func(reflect.Value, ast.Value, *decoder) error

var decodeCache = makeCache[reflect.Type, decodeFunc]()

func decode(rd *reader, ctx *Context, v any, flgs ...klonflags.Flags) (err error) {
	defer handlePanic(&err)
	doc, errs := rd.parseDocument()
	if len(errs) > 0 {
		return errs[0]
	}
	return decodeDocument(doc, ctx, v, flgs...)
}

func decodeDocument(doc *ast.Document, ctx *Context, v any, flgs ...klonflags.Flags) error {
	rv := reflect.ValueOf(v)
	// Make sure v is a non-nil pointer
	rt := rv.Type()
	switch {
	case rt == nil:
		return errors.New("klon: nil argument passed to Unmarshall")
	case rv.Kind() != reflect.Pointer:
		return fmt.Errorf("klon: non-pointer type %s passed to Unmarshall", rt.String())
	case rv.IsNil():
		return fmt.Errorf("klon: nil %s passed to Unmarshall", rt.String())
	}

	d := &decoder{
		ctx:   ctx,
		vars:  doc.Variables,
		flags: parseFlags(flgs...),
	}
	return d.decodeValue(doc.Body, rv.Elem())
}

func (d *decoder) decodeValue(val ast.Value, rv reflect.Value) error {
	decode := d.getDecoder(rv.Type())
	return decode(rv, val, d)
}

// Looks up a decoder or creates one if it doesn't exist.
func (d *decoder) getDecoder(rt reflect.Type) decodeFunc {
	if decode, ok := decodeCache.get(rt); ok {
		return decode
	}
	// If the type implements [Unmarshaller] or [encoding.TextUnmarshaler], use their decoder.
	var decode decodeFunc
	ptr := reflect.PointerTo(rt)
	switch {
	case rt.Implements(unmarshallerType), ptr.Implements(unmarshallerType):
		decode = decodeUnmarshaller
	case rt.Implements(textUnmarshalerType), ptr.Implements(textUnmarshalerType):
		decode = decodeTextUnmarshaller
	default:
		decode = d.makeDefaultDecoder(rt)
	}

	decode = preprocessValue(decode)
	decodeCache.set(rt, decode)
	return decode
}

func typeMismatchError(rv reflect.Value, val ast.Value) *Error {
	goType := rv.Type()
	var msg string
	if rv.Kind() == reflect.Interface {
		msg = "Can't use " + FormatNodeType(val) + " as a value"
	} else {
		msg = "Expected " + formatGoType(goType) + ", but found " + FormatNodeType(val)
	}
	return &Error{
		Code:        klonerrs.ErrTypeMismatch,
		Type:        goType,
		Value:       val,
		Range:       val.Pos(),
		Text:        msg,
		isDecodeErr: true,
	}
}

func formatGoType(rt reflect.Type) string {
	switch k := rt.Kind(); k {
	case reflect.String:
		return "a string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float64, reflect.Float32:
		return "a number"
	case reflect.Bool:
		return "a boolean"
	case reflect.Slice, reflect.Array:
		return "a list"
	case reflect.Map, reflect.Struct:
		return "an object"
	case reflect.Pointer:
		return formatGoType(rt.Elem())
	default:
		return k.String()
	}
}

func FormatNodeType(node ast.Value) string {
	switch node.(type) {
	case *ast.String, *ast.StringGroup:
		return "a string"
	case *ast.Number:
		return "a number"
	case *ast.Boolean:
		return "a boolean"
	case *ast.List:
		return "a list"
	case *ast.Object:
		return "an object"
	case *ast.None:
		return "no value"
	default:
		panic(fmt.Sprintf("unhandled node type: %T", node))
	}
}

func decodeError(code klonerrs.Code, rv reflect.Value, val ast.Value,
	msg string, v ...any,
) *Error {
	var errMsg string
	if len(v) > 0 {
		errMsg = fmt.Sprintf(msg, v...)
	} else {
		errMsg = msg
	}
	var rt reflect.Type
	if rv.IsValid() {
		rt = rv.Type()
	}
	var pos ranges.Range
	if val != nil {
		pos = val.Pos()
	}
	return &Error{
		Code:        code,
		Text:        errMsg,
		Type:        rt,
		Value:       val,
		Range:       pos,
		isDecodeErr: true,
	}
}

func (d *decoder) warn(e error) {
	w, ok := e.(*Error)
	if ok {
		w.Warning = true
		d.ctx.Warnings = append(d.ctx.Warnings, w)
	}
}

func (d *decoder) shouldWarn(err error) bool {
	e, ok := err.(*Error)
	if !ok || d.ctx == nil || d.ctx.WarningKinds == nil || d.ctx.Warnings == nil {
		return false
	}
	_, ok = d.ctx.WarningKinds[e.Code]
	return ok
}

func (d *decoder) valueToString(v ast.Value) (string, error) {
	switch v := v.(type) {
	case *ast.String:
		return d.evaluateString(v)
	case *ast.Boolean:
		return v.String(), nil
	case *ast.Number:
		return v.Source, nil
	default:
		return "", decodeError(klonerrs.ErrCantConvertToString, reflect.Value{}, v,
			"Can't convert %s to a string", FormatNodeType(v),
		)
	}
}
