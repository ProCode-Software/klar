package decode

import (
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/context"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/reader"
)

type Decoder struct {
	ctx   *context.Context
	flags flags.Flags
}

func Decode(rd *reader.Reader, ctx *context.Context, v any, flgs ...flags.Flags) error {
	doc, errs := rd.ParseDocument()
	if len(errs) > 0 {
		return errs[0]
	}
	return DecodeDocument(doc, ctx, v, flgs...)
}

func DecodeDocument(doc *ast.Document, ctx *context.Context, v any, flgs ...flags.Flags) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &errors.InvalidUnmarshallError{Type: rv.Type()}
	}
	d := &Decoder{
		ctx:   ctx,
		flags: flags.Parse(flgs...),
	}
	return d.DecodeValue(doc.Body, rv.Elem())
}

func (d *Decoder) DecodeValue(node ast.Node, v reflect.Value) error {
	/* marsh := d.lookupDecodeFunc(rt)
	body, err := marsh(rv, d, topLevel)
	if err != nil {
		return err
	} */
	return nil
}
