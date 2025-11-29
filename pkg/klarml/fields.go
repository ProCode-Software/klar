package klarml

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type StructFields struct {
	Flat         []*StructField          // Sorted fields
	Fields       map[string]*StructField // In lower case
	ByActualCase map[string]*StructField
}

type StructField struct {
	Name    string
	Decode  decodeFunc
	Encode  any
	Type    reflect.Type
	Indices []int
}

func makeStructFields(rt reflect.Type, flag Flags) (StructFields, error) {
	type queueItem struct {
		Type reflect.Type
	}
	var (
		visited    = map[reflect.Type]struct{}{}
		currFields []*StructField
		nextFields = []*StructField{{Type: rt}}
		fieldLen   = rt.NumField()
		strFields  = StructFields{
			Flat:         make([]*StructField, 0, fieldLen),
			Fields:       make(map[string]*StructField, fieldLen),
			ByActualCase: make(map[string]*StructField, fieldLen),
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
						continue
					}
				} else if !f.IsExported() {
					continue
				}
				var name string
				name, ok := f.Tag.Lookup("klarml")
				// Check for json: struct tag and extract the name only
				if !ok && flag.Has(AllowJSONStructTags) {
					if parts := strings.Split(f.Tag.Get("json"), ","); len(parts) > 0 {
						name = parts[0]
					}
				}
				if name == "-" {
					continue
				}
				indices := make([]int, len(field.Indices)+1)
				copy(indices, field.Indices)
				indices[len(field.Indices)] = i

				rt := f.Type
				if rt.Name() == "" && rt.Kind() == reflect.Pointer {
					rt = rt.Elem()
				}
				isNormalField := name != "" || !f.Anonymous || rt.Kind() != reflect.Struct
				if isNormalField || flag.Has(KeyedEmbeddedFields) {
					if name == "" {
						name = f.Name
					}
					new := &StructField{
						Name:    name,
						Type:    rt,
						Indices: indices,
					}
					lower := strings.ToLower(name)
					strFields.Flat = append(strFields.Flat, new)
					if existing, ok := strFields.Fields[lower]; ok {
						// If the existing one is a direct field, keep it. Otherwise,
						// a name shared by two embedded structs is an error. This
						// is similar to Go's behaviour.
						if !isNormalField && len(existing.Indices) > 1 {
							return strFields,
								fmt.Errorf("ambiguous use of field: %s", name)
						}
						// Otherwise, don't reassign it
					} else {
						strFields.Fields[lower] = new
						strFields.ByActualCase[name] = new
					}
					// Continue unless it is an embedded field
					if isNormalField {
						continue
					}
				}
				// Embedded struct
				nextFields = append(nextFields, &StructField{
					Name:    name,
					Type:    rt,
					Indices: indices,
				})
			}
		}
	}
	slices.SortFunc(strFields.Flat, func(a, b *StructField) int {
		return cmp.Or(
			strings.Compare(a.Name, b.Name),
			len(a.Indices)-len(b.Indices),
		)
	})
	return strFields, nil
}
