package klon

import (
	"reflect"
	"strings"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

// preprocessValue wraps a new [decodeFunc] that resolves variables
// and concatenates strings before decoding.
func preprocessValue(decode decodeFunc) decodeFunc {
	return func(rv reflect.Value, val ast.Value, d *decoder) (err error) {
		switch node := val.(type) {
		case *ast.None:
			if d.flags.Has(klonflags.ZeroNullValues) {
				rv.SetZero()
			}
			return nil // No need to decode
		case *ast.VarRef:
			val, err = d.resolveVar(node)
			// Preprocess the resolved value (should not be a VarRef)
			if err == nil {
				return preprocessValue(decode)(rv, val, d)
			}
		case *ast.StringGroup:
			val, err = d.preprocessStringGroup(node)
		case *ast.Object:
			val, err = d.preprocessObject(node)
		case *ast.ArrowRef:
			// ArrowRefs are invalid outside lists (manually handled) and Object (*Field.Arrow)
			return decodeError(klonerrs.ErrMisplacedRest, reflect.Value{}, node,
				"Rests are only allowed in objects and lists",
			)
		}
		if err != nil {
			return err
		}
		return decode(rv, val, d)
	}
}

func (d *decoder) resolveVar(ref *ast.VarRef) (ast.Value, error) {
	origRef := ref
	var chain []string // Excludes the original reference

	if d.vars == nil {
		goto notFound
	}
	for {
		v, ok := d.vars[ref.Name]
		if !ok {
			goto notFound
		}
		ref, ok = v.(*ast.VarRef)
		if !ok {
			// Set the values of the original variable and its dependencies to
			// the resolved value for performance.
			d.vars[origRef.Name] = v
			for _, name := range chain {
				d.vars[name] = v
			}
			return v, nil
		}

		// Variable declaration uses another variable. Continue resolving.
		chain = append(chain, ref.Name)

		if ref.Name == origRef.Name {
			// Circular reference detected
			if len(chain) > 1 {
				return nil, decodeError(klonerrs.ErrVarCycle, reflect.Value{}, ref,
					"Variable '%s' refers to itself in a cycle (%[1]s -> %s)",
					ref.Name, strings.Join(chain, " -> "),
				)
			} else {
				return nil, decodeError(klonerrs.ErrVarCycle, reflect.Value{}, ref,
					"Variable '%s' is defined in terms of itself", ref.Name,
				)
			}
		}
	}

notFound:
	return nil, decodeError(klonerrs.ErrUndefinedVar, reflect.Value{}, ref,
		"Can't find variable '%s'", ref.Name,
	)
}

func (d *decoder) preprocessStringGroup(sg *ast.StringGroup) (*ast.String, error) {
	// TODO: resolve classes and maybe concatenate
	return &ast.String{}, nil
}

// preprocessObject resolves rest items and merges them with the object.
func (d *decoder) preprocessObject(obj *ast.Object) (*ast.Object, error) {
	var new *ast.Object
	for i, f := range obj.Fields {
		if f.Arrow == nil {
			// Fields are only duplicated if there are arrow references
			if new != nil {
				new.Fields = append(new.Fields, f)
			}
			continue
		}

		// Rest
		if new == nil {
			new = &ast.Object{
				BaseNode: obj.BaseNode,
				Inline:   obj.Inline,
				Fields:   make([]*ast.Field, 0, len(obj.Fields)),
			}
			new.Fields = append(new.Fields, obj.Fields[:i]...)
		}
		// Resolve the arrow reference
		v, err := d.resolveVar(f.Arrow.Var)
		if err != nil {
			return nil, err
		}
		obj, ok := v.(*ast.Object)
		if !ok {
			if _, ok := v.(*ast.None); ok {
				continue // Rest can be 'none'
			}
			return nil, decodeError(klonerrs.ErrInvalidRest, reflect.Value{}, f.Arrow.Var,
				"'%s' must be an object in order to use it as a rest", f.Arrow.Var.Name,
			)
		}
		_ = obj
		// TODO: check v and append its fields
	}
	if new == nil {
		return obj, nil
	}
	return new, nil
}
