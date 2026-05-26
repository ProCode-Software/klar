package klon

import (
	"reflect"
	"strings"

	"github.com/ProCode-Software/klar/internal/klarerrs"
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
			if rv.Kind() == reflect.Interface && d.flags.Has(klonflags.EmptyValueIsString) {
				rv.Set(reflect.ValueOf(""))
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
	err := decodeError(klonerrs.ErrUndefinedVar, reflect.Value{}, ref,
		"Can't find variable '%s'", ref.Name,
	)
	if d.shouldWarn(err.Code) {
		d.warn(err)
		return &ast.None{}, nil
	}
	return nil, err
}

func resolveRest[T ast.Value](d *decoder, rest *ast.ArrowRef, rv reflect.Value) (val T, empty bool, err error) {
	res, err := d.resolveVar(rest.Var)
	if err != nil {
		return val, false, err
	}
	val, ok := res.(T)
	if !ok {
		if _, ok := res.(*ast.None); ok {
			return val, true, nil // Rest can be none
		}
		var kind string
		switch res.(type) {
		case *ast.List:
			kind = "a list"
		case *ast.Object:
			kind = "an object"
		}
		return val, false, decodeError(klonerrs.ErrInvalidRest, rv, res,
			"'%s' must be %s in order to use it as a rest", rest.Var.Name, kind,
		)
	}
	return val, false, nil
}

func (d *decoder) preprocessStringGroup(sg *ast.StringGroup) (*ast.String, error) {
	// TODO: resolve classes and maybe concatenate
	return &ast.String{}, nil
}

// preprocessObject resolves rest items and merges them with the object.
func (d *decoder) preprocessObject(obj *ast.Object) (*ast.Object, error) {
	// 1. Check for duplicate fields and whether there are rests
	var (
		literalKeys = make(map[string]*ast.Field)
		hasRest     bool
	)
	for _, f := range obj.Fields {
		if f.Arrow != nil {
			hasRest = true
			continue
		}
		path, err := d.stringFieldPath(f)
		if err != nil {
			return nil, err
		}
		if existing, ok := literalKeys[path]; ok {
			return nil, decodeError(klonerrs.ErrDuplicateField, reflect.Value{}, f,
				"Field %s was already defined at %s", klarerrs.Quote(path), existing.Pos(),
			)
		}
		literalKeys[path] = f
	}
	// Don't duplicate the object if there aren't any rests
	if !hasRest {
		return obj, nil
	}

	// 2. Merge rest newFields into the object. Fields maintain the
	// order they were first defined in.
	var (
		newFields []*ast.Field
		seen      = make(map[string]int) // Key: Index
		addField  func(*ast.Field) error
	)
	addField = func(f *ast.Field) error {
		if f.Arrow == nil {
			// Normal field or key-path. Key-paths will be resolved during decoding.
			// For now, we only verify that the same key-path doesn't appear twice,
			// not that `a.b: x` and `a: {b: x}` are both present.
			path, err := d.stringFieldPath(f)
			if err != nil {
				return err
			}
			if i, ok := seen[path]; ok {
				// Replace existing field (original order, but last value)
				newFields[i] = f
			} else {
				// New field
				seen[path] = len(newFields)
				newFields = append(newFields, f)
			}
			return nil
		}

		// Rest: Resolve and recursively preprocess
		restObj, empty, err := resolveRest[*ast.Object](d, f.Arrow, reflect.Value{})
		if err != nil {
			return err
		} else if empty {
			return nil
		}
		// Also preprocess the rest target object to resolve nested rests
		if restObj, err = d.preprocessObject(restObj); err != nil {
			return err
		}
		// Add all fields from the rest
		for _, subField := range restObj.Fields {
			if err := addField(subField); err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range obj.Fields {
		if err := addField(f); err != nil {
			return nil, err
		}
	}

	return &ast.Object{
		BaseNode: obj.BaseNode,
		Inline:   obj.Inline,
		Fields:   newFields,
	}, nil
}

func (d *decoder) evaluateString(str *ast.String) (string, error) {
	if str.Evaluated != "" {
		return str.Evaluated, nil
	}
	return str.Raw, nil
}

// stringFieldPath returns the string representation of a field's key.
// It handles both keys and key paths.
func (d *decoder) stringFieldPath(f *ast.Field) (string, error) {
	if f.KeyPath == nil {
		return d.ToString(f.Key)
	}
	var b strings.Builder
	for _, p := range *f.KeyPath {
		keyStr, err := d.ToString(p)
		if err != nil {
			return "", err
		}
		b.WriteByte('.')
		b.WriteString(keyStr)
	}
	return b.String()[1:], nil // Cut dot at beginning
}
