package klon

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

type Unmarshaller interface {
	UnmarshalKlarMarkup(ast.Node) error
}

func tryUnmarshaller(rv reflect.Value, data ast.Node) (ok bool, err error) {
	if !rv.Elem().CanInterface() {
		return false, nil
	}
	if u, ok := rv.Interface().(Unmarshaller); ok {
		return true, u.UnmarshalKlarMarkup(data)
	}
	return false, nil
}

type goIface struct {
	typ unsafe.Pointer
	ptr unsafe.Pointer
}

func UnmarshallDocument(doc *ast.Document, dst any, flags ...Flags) error {
	flag := parseFlags(flags)
	rt := reflect.TypeOf(dst)
	ptr := (*goIface)(unsafe.Pointer(&dst)).ptr

	if rt == nil || ptr == nil || rt.Kind() != reflect.Pointer {
		return unmarshallDstError(rt)
	}
	rt = rt.Elem()

	/* const prefix = "klon.Unmarshall: "
	rv := reflect.ValueOf(dst)
	rt := reflect.TypeOf(dst)
	switch {
	case rv.Kind() != reflect.Pointer:
		return errors.New("klon.Unmarshall(data, dst): dst must be a pointer")
	case rv.IsNil():
		return errors.New("klon.Unmarshall(data, dst): dst must not be nil")
	}
	if !rv.IsValid() {
		return fmt.Errorf("klon.Unmarshall: %v is not a valid reflect.Value", rv)
	}
	ctx := NewContext(doc)
	errors := ctx.ResolveVars()
	if len(errors) > 0 {
		return fmt.Errorf("klon.Unmarshall: %w", errors[0])
	}
	ok, err := tryUnmarshaller(rv, doc.Body)
	if ok && err != nil {
		return err
	}
	elem := rv.Elem()
	typeElem := rt.Elem()
	switch node := doc.Body.(type) {
	case Object:

	case StringLiteral:
		if elem.Kind() != reflect.String {
			return fmt.Errorf(prefix+"cannot unmarshall string into non-string %T", rv)
		}
	case NumericLiteral:
		if !elem.CanFloat() {
			return fmt.Errorf(prefix+"cannot unmarshall number into non-numeric type %T", rv)
		}
	} */
	return nil
}

func Unmarshall(data []byte, dst any, flags ...Flags) error {
	doc, err := Parse(data)
	if len(err) > 0 {
		return fmt.Errorf("klon.Unmarshall: parse error: %w", err[0])
	}
	return UnmarshallDocument(doc, dst, flags...)
}

func parseFlags(flags []Flags) (f Flags) {
	for _, flag := range flags {
		f |= flag
	}
	return f
}
