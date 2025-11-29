package klarml

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

type decoder struct {
	ctx   *Context
	flags Flags
	vars  map[string]ast.Value
}

type decodeFunc func(reflect.Value, ast.Value, *decoder) error

var DecodeCache = MakeCache[reflect.Type, decodeFunc]()

func decode(rd *reader, ctx *Context, v any, flgs ...Flags) (err error) {
	defer handlePanic(&err)
	doc, errs := rd.parseDocument()
	if len(errs) > 0 {
		return errs[0]
	}
	return decodeDocument(doc, ctx, v, flgs...)
}

func decodeDocument(doc *ast.Document, ctx *Context, v any, flgs ...Flags) error {
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
	if marsh, ok := DecodeCache.Get(rt); ok {
		return marsh
	}
	marsh := d.makeDefaultDecoder(rt)
	DecodeCache.Set(rt, marsh)
	return marsh
}

func typeError(rv reflect.Value, val ast.Value) *TypeError {
	rt := rv.Type()
	// Known pointer
	nodeType := reflect.TypeOf(val).Elem().Name()
	return &TypeError{
		Code: ErrTypeMismatch,
		Type: rt,
		Val:  val,
		Text: "can't decode " + nodeType + " into Go type " + rt.Name(),
	}
}

func decodeError(code ErrorCode, rv reflect.Value, val ast.Value, msg string, v ...any) error {
	var errMsg string
	if len(v) > 0 {
		errMsg = fmt.Sprintf(msg, v...)
	} else {
		errMsg = msg
	}
	return &TypeError{Code: code, Type: rv.Type(), Val: val, Text: errMsg}
}
