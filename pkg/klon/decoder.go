package klon

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
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
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshallError{Type: rv.Type()}
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
	if marsh, ok := decodeCache.get(rt); ok {
		return marsh
	}
	marsh := d.makeDefaultDecoder(rt)
	decodeCache.set(rt, marsh)
	return preprocessValue(marsh)
}

// preprocessValue wraps a new [decodeFunc] that resolves variables
// and concatenates strings before decoding.
func preprocessValue(decode decodeFunc) decodeFunc {
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		switch node := val.(type) {
		case *ast.VarRef:
			if v, ok := d.vars[node.Name]; ok {
				val = v
				break
			}
			return decodeError(klonerrs.ErrUndefinedVar, rv, node,
				"Can't find variable '%s'", node.Name,
			)
		case *ast.StringGroup:
			// TODO: resolve classes
		}
		return decode(rv, val, d)
	}
}

func typeMismatchError(rv reflect.Value, val ast.Value) *Error {
	goType := rv.Type()
	var msg string
	if rv.Kind() == reflect.Interface {
		msg = "Can't use " + FormatNodeType(val) + " as a value"
	} else {
		msg = "Expected " + formatGoType(goType.Kind()) + ", but found " + FormatNodeType(val)
	}
	return &Error{
		Code:  klonerrs.ErrTypeMismatch,
		Type:  goType,
		Value: val,
		Range: val.Pos(),
		Text:  msg,
	}
}

func formatGoType(k reflect.Kind) string {
	switch k {
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
) error {
	var errMsg string
	if len(v) > 0 {
		errMsg = fmt.Sprintf(msg, v...)
	} else {
		errMsg = msg
	}
	return &Error{
		Code:  code,
		Text:  errMsg,
		Type:  rv.Type(),
		Value: val,
		Range: val.Pos(),
	}
}
