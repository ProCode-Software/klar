package decode

import (
	"reflect"
	"sync"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

func (d *Decoder) makeArrayDecoder(rt reflect.Type) decodeFunc {
	length, itemRT := rt.Len(), rt.Elem()
	decodeItem := d.lookupDecodeFunc(itemRT)
	return func(v reflect.Value, d *Decoder, pf parseFlags) (_ ast.Node, err error) {
		// TODO: support for arrays of bytes/runes (decode as string)
		if err = d.Expect('['); err != nil {
			return
		}
		var i int
		for ; ; i++ {
			if err = d.SkipSpaceNewline(); err != nil {
				return
			}
			if i < length {
				if _, err = decodeItem(v.Index(i), d, pf|comma); err != nil {
					return
				}
			} else if _, err = d.ReadValue(pf | comma); err != nil {
				// Too many items, but ignore the rest
				return
			}
			if err = d.SkipSpaceNewline(); err != nil {
				return
			}
			if d.Curr() == ']' {
				break
			} else if err = d.Expect(','); err != nil {
				return
			}
		}
		if err = d.Expect(']'); err != nil && err != EOF {
			return
		}
		checkEOF(&err)
		i += 1
		if i != length && !d.Flags.Has(flags.IgnoreArrayLength) {
			err = &errors.InvalidArrayLengthError{Need: length, Got: i}
		}
		v.Index(i)
		return
	}
}

// TODO
func (d *Decoder) makeSliceDecoder(rt reflect.Type) decodeFunc {
	var (
		itemRT     = rt.Elem()
		decodeItem = d.lookupDecodeFunc(itemRT)
		itemPool   = sync.Pool{New: func() any {
			return reflect.New(itemRT).Elem()
		}}
	)
	_, _ = decodeItem, &itemPool
	return nil
}
