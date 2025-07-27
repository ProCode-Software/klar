package decode

import (
	goerrors "errors"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

var ErrUnterminatedObject = goerrors.New("expected '}' to close object")

// TODO: make this a struct error
var ErrInvalidDepth = goerrors.New("too much depth")
var ErrUnknownField = goerrors.New("unknown field")

func (d *Decoder) readKey() (path []string, depth int, err error) {
	if err := d.SkipSpaceNewline(); err != nil {
		return nil, 0, err
	}
	for d.Curr() == '-' {
		depth++
		if _, err := d.Advance(); err != nil {
			// Invalid syntax: - and then EOF
			return nil, depth, err
		}
		if err := d.SkipSpace(); err != nil {
			// Same
			return nil, depth, err
		}
	}
	if depth > d.Depth {
		// Too much depth
		return nil, depth, ErrInvalidDepth
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
				// Nil pointer
				/* src: sonnet
				elm := val.Type().Elem()
				if !val.CanSet() {
					return reflect.Value{}, fieldError(tmpl + elm.String())
				}
				val.Set(reflect.New(elm))
				*/
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

func (d *Decoder) makeStructDecoder(rt reflect.Type) decodeFunc {
	fields, _ := makeStructFields(rt, d.Flags)
	return func(v reflect.Value, d *Decoder) (ast.Node, error) {
		if err := d.SkipSpaceNewline(); err != nil {
			// Nothing to read if EOF
			return nil, err
		}
		obj := &ast.Object{
			Props: make([]*ast.Prop, 0, len(fields.Flat)),
		}
		// Keeping separate maps because keypath slice can't be stored as a key,
		// and we don't want to convert it to a string because the existing field
		// could be a string.
		exist := make(map[string]struct{}, len(fields.Flat))
		var existPath [][]string
		// Object literal
		if d.Curr() == '{' {
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
			var rv reflect.Value
			if len(path) > 1 {
				existPath = append(existPath, path)
			} else {
				exist[prop.Key] = struct{}{}
				if field, ok := fields.Fields[prop.Key]; ok {
					rv = v.FieldByIndex(field.Index)
				} else if d.Flags.Has(flags.NoUnknownFields) {
					return obj, ErrUnknownField
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

		}
		return obj, nil
	}
}
