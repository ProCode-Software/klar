package klon

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

type structFields struct {
	Flat         []*structField          // Sorted fields
	FoldedFields map[string]*structField // In lower case
	Fields       map[string]*structField // Actual case (camel case or Go case)
}

type structField struct {
	Name   string
	Decode decodeFunc // Value decoder
	Encode any        // TODO
	Type   reflect.Type
	// Path to reach the actual field when embedded. If the field is not embedded,
	// len(Indices) == 1. Each index is passed to [reflect.Value.FieldByIndex].
	Indices []int
}

func makeStructFields(rt reflect.Type, flag klonflags.Flags) (structFields, error) {
	var (
		visited    = map[reflect.Type]struct{}{}
		currFields []*structField
		nextFields = []*structField{{Type: rt}}

		fieldLen  = rt.NumField()
		strFields = structFields{
			Flat:         make([]*structField, 0, fieldLen),
			Fields:       make(map[string]*structField, fieldLen),
			FoldedFields: make(map[string]*structField, fieldLen),
		}
	)
	// Breadth-first search
	for len(nextFields) > 0 {
		currFields, nextFields = nextFields, currFields[:0]
		for _, field := range currFields {
			if _, ok := visited[field.Type]; ok {
				continue
			}
			visited[field.Type] = struct{}{}

			for i := range field.Type.NumField() {
				f := field.Type.Field(i)
				if f.Anonymous {
					rt := f.Type
					if rt.Kind() == reflect.Pointer {
						rt = rt.Elem()
					}
					if !f.IsExported() && rt.Kind() != reflect.Struct {
						// Unexported non-struct embedded field
						continue
					}
				} else if !f.IsExported() {
					continue // Unexported fields aren't settable in Go
				}

				// Check struct tags for name
				name, ok := f.Tag.Lookup("klon")
				// Check for json: struct tag and extract the name (1st before comma) only
				if !ok && flag.Has(klonflags.AllowJSONStructTags) {
					for name = range strings.SplitSeq(f.Tag.Get("json"), ",") {
						break
					}
				}
				if name == "-" {
					continue // Don't include this field
				}

				// Check for "options" struct tag
				var decode decodeFunc
				opts, ok := f.Tag.Lookup("options")
				if ok {
					decode = preprocessValue(makeEnumDecoder(opts))
				}

				indices := make([]int, len(field.Indices)+1)
				copy(indices, field.Indices)
				indices[len(field.Indices)] = i

				rt := f.Type
				if rt.Name() == "" && rt.Kind() == reflect.Pointer {
					rt = rt.Elem()
				}
				isNormalField := name != "" || !f.Anonymous || rt.Kind() != reflect.Struct
				if isNormalField || flag.Has(klonflags.KeyedEmbeddedFields) {
					if name == "" {
						// No Klon struct field: use a default name
						if flag.Has(klonflags.CaseSensitiveFields)  {
							name = f.Name
						} else {
							// Klon converts field names to camel case by default
							name = camelCaseField(f.Name, flag)
						}
					}
					new := &structField{
						Name:    name,
						Type:    rt,
						Indices: indices,
						Decode:  decode,
					}
					lower := strings.ToLower(name)
					strFields.Flat = append(strFields.Flat, new)
					if _, ok := strFields.Fields[lower]; ok {
						// Could be caused by:
						// - 2 fields with same struct tag name
						// - FieldA `klon:Field_A` and Field_A
						// - Embedded field and non-embedded field with same name
						// TODO: Check if there are exceptions and precedence rules
						// to avoid returning an error
						return strFields, fmt.Errorf("duplicate field: %s", name)
					} else {
						strFields.Fields[name] = new
						strFields.FoldedFields[lower] = new
					}
					// Continue unless it is an embedded field (can be either keyed
					// or unkeyed because KeyedEmbeddedFields is on)
					if isNormalField {
						continue
					}
				}
				// Embedded struct
				nextFields = append(nextFields, &structField{
					Name:    name,
					Type:    rt,
					Indices: indices,
					Decode:  decode,
				})
			}
		}
	}
	if flag.Has(klonflags.NoSortFields) {
		return strFields, nil // Don't sort fields
	}
	slices.SortFunc(strFields.Flat, func(a, b *structField) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), len(a.Indices)-len(b.Indices))
	})
	return strFields, nil
}

func camelCaseField(name string, flags klonflags.Flags) string {
	if flags.Has(klonflags.PreserveFieldCase) {
		return name
	}
	var numUpper int
	for _, c := range name {
		if unicode.IsLower(c) {
			if numUpper > 1 {
				numUpper--
			}
			break
		}
		numUpper++
	}
	return strings.ToLower(name[:numUpper]) + name[numUpper:]
}

// TODO: Enum decoder for slice/array items and map values
func makeEnumDecoder(optsKey string) decodeFunc {
	return func(rv reflect.Value, val ast.Value, d *decoder) error {
		if d.ctx == nil || d.ctx.Enums == nil {
			return fmt.Errorf("enum key %s doesn't exist", optsKey)
		}
		opts := d.ctx.Enums[optsKey]
		var valAsStr string
		switch val := val.(type) {
		case *ast.String:
			valAsStr = val.Raw
		case *ast.Number:
			valAsStr = val.Source
		case *ast.Boolean:
			valAsStr = strconv.FormatBool(val.Value)
		default:
			allOpts := strings.Join(slices.Sorted(maps.Keys(opts)), ", ")
			return decodeError(klonerrs.ErrInvalidEnumOption, rv, val,
				"Invalid option: Expected one of: %s", allOpts,
			)
		}
		enumVal, ok := opts[valAsStr]
		if !ok {
			allOpts := strings.Join(slices.Sorted(maps.Keys(opts)), ", ")
			return decodeError(klonerrs.ErrInvalidEnumOption, rv, val,
				"Invalid option %s: Expected one of: %s", klarerrs.Quote(valAsStr), allOpts,
			)
		}
		rv.Set(reflect.ValueOf(enumVal))
		return nil
	}
}

func (sf *structFields) Lookup(name string, val ast.Value, flag klonflags.Flags) (
	*structField, error,
) {
	if field, ok := sf.Fields[name]; ok {
		return field, nil
	}
	if !flag.Has(klonflags.CaseSensitiveFields) {
		// Case insensitive lookup
		if field, ok := sf.FoldedFields[strings.ToLower(name)]; ok {
			return field, nil
		}
	}
	// TODO: Recommend a similar field name, or hint about whitespace ('a ' vs. 'a')
	return nil, decodeError(klonerrs.ErrFieldNotFound, reflect.Value{}, val,
		"Unknown field '%s'", name,
	)
}
