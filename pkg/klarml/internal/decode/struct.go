package decode

import (
	"cmp"
	goerrors "errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

var (
	ErrUnterminatedObject = goerrors.New("expected '}' to close object")
	ErrTooManyDash        = goerrors.New("too many '-'")
)

func (d *Decoder) readKey() (path []string, depth int, err error) {
	if err := d.SkipSpaceNewline(); err != nil {
		return nil, 0, err
	}
	for d.Curr() == '-' {
		depth++
		if err := cmp.Or(d.Expect('-'), d.SkipSpace()); err != nil {
			// Invalid syntax: - and then EOF
			return nil, depth, err
		}
	}
	if depth > d.Depth {
		// Too much depth
		return nil, depth, ErrTooManyDash
	}
	var isVar bool
	// The key or path
	if d.Curr() == '$' {
		isVar = true
	}
	for {
		// String key
		if curr := d.Curr(); curr == '"' || curr == '\'' {
			str, err := d.readString()
			if err != nil {
				return path, depth, err
			}
			path = append(path, str.Value)
		} else {
			key, err := d.ReadIdent()
			if err != nil {
				return path, depth, err
			}
			path = append(path, key)
		}
		// Space after key
		if err := d.SkipSpace(); err != nil {
			return path, depth, err
		}
		if isVar || d.Curr() != '/' {
			break
		}
		// Skip space after / to read next path
		if err = d.SkipSpace(); err != nil {
			return
		}
	}
	return
}

func followValue(v reflect.Value, indices []int) (reflect.Value, error) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for _, i := range indices {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				// Initialize pointer
				e := v.Type().Elem()
				if !v.CanSet() {
					return reflect.Value{}, goerrors.New("cannot set reflect.Value")
				}
				v.Set(reflect.New(e))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

func (d *Decoder) GetField(fields StructFields, name string) (*StructField, bool) {
	m := fields.Fields
	if d.Flags.Has(flags.CaseSensitiveFields) {
		m = fields.ByActualCase
	}
	field, ok := m[name]
	return field, ok
}

func (d *Decoder) makeStructDecoder(rt reflect.Type) decodeFunc {
	fields, _ := makeStructFields(rt, d.Flags)
	return func(v reflect.Value, d *Decoder) (ast.Node, error) {
		if err := d.SkipSpaceNewline(); err != nil {
			// Nothing to read if EOF
			return nil, err
		}
		obj := &ast.Object{
			Props: make([]*ast.Prop, 0, len(fields.Fields)),
		}
		// Keeping separate maps because keypath slice can't be stored as a key,
		// and we don't want to convert it to a string because the existing field
		// could be a string.
		var (
			exist     = make(map[string]struct{}, len(fields.Flat))
			existPath [][]string
			once      sync.Once
			seps      = []byte{'\n'}
		)
		once.Do(func() {
			for _, field := range fields.Fields {
				if field.Decode == nil {
					field.Decode = d.lookupMarshallFunc(field.Type)
				}
			}
		})
		// Object literal
		if d.Curr() == '{' {
			oldDepth := d.Depth
			oldComma := d.CommaSep
			defer func() {
				d.Depth = oldDepth
				d.CommaSep = oldComma
			}()
			d.Depth = 0
			d.CommaSep = true
			seps = append(seps, ',', '}')
			if _, err := d.Advance(); err != nil {
				if err == EOF {
					return nil, ErrUnterminatedObject
				}
				return nil, err
			}
			obj.Inline = true
		}
		for {
			prop := &ast.Prop{}
			if obj.Inline && d.Curr() == '}' {
				if _, err := d.Advance(); err != nil {
					checkEOF(&err)
					return obj, err
				}
				break
			}
			path, depth, err := d.readKey()
			if err != nil {
				return obj, err
			}
			if depth < d.Depth {
				break
			}
			prop.Path = path
			prop.Key = path[len(path)-1]
			var (
				rv    reflect.Value
				field *StructField
			)
			if len(path) > 1 {
				existPath = append(existPath, path)
			} else {
				key := prop.Key
				var ok bool
				if _, ok = exist[key]; ok {
					return obj, goerrors.New("duplicate field: " + key)
				}
				exist[key] = struct{}{}
				if field, ok = d.GetField(fields, key); ok {
					rv, err = followValue(v, field.Indices)
					if err != nil {
						return obj, err
					}
				} else if d.Flags.Has(flags.NoUnknownFields) {
					return obj, goerrors.New("unknown field: " + key)
				} else {
					// Unknown field: read the value
					continue
				}
			}
			if err = d.Expect(':'); err != nil {
				return obj, err
			}
			// Skip space after :
			if err = d.SkipSpace(); err != nil {
				if err == EOF {
					// Null
					prop.Value = &ast.Null{}
					obj.Props = append(obj.Props, prop)
					return obj, nil
				}
				return obj, err
			}
			// Read a value
			d.Depth++
			valNode, err := field.Decode(rv, d)
			if valNode != nil {
				prop.Value = valNode.(ast.Value)
			}
			d.Depth--
			obj.Props = append(obj.Props, prop)
			// Error above
			if err != nil {
				fmt.Printf("%#v\n", valNode)
				return obj, err
			}
			// Decode does not return error when at EOF.
			if d.Overflow() {
				break
			}
			// Expect a newline
			if err := cmp.Or(
				d.SkipSpace(), d.ExpectOne(seps...), d.SkipSpaceNewline(),
			); err != nil {
				checkEOF(&err)
				return obj, err
			}
		}
		return obj, nil
	}
}
