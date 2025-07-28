package decode

import (
	"cmp"
	"reflect"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type StructFields struct {
	Flat   []*StructField
	Fields map[string]*StructField
}

type StructField struct {
	Name    string
	Decode  decodeFunc
	Encode  any
	Type    reflect.Type
	Indices []int
}

func makeStructFields(rt reflect.Type, flag flags.Flags) (StructFields, error) {
	type queueItem struct {
		Type reflect.Type
	}
	var (
		visited    = map[reflect.Type]struct{}{}
		currFields []*StructField
		nextFields = []*StructField{{Type: rt}}
		fieldLen = rt.NumField()
		strFields  = StructFields{
			Flat: make([]*StructField, 0, fieldLen),
			Fields: make(map[string]*StructField, fieldLen),
		}
	)
	lowerName := func(name string) string {
		if flag.Has(flags.CaseSensitiveFields) {
			return name
		}
		return strings.ToLower(name)
	}
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
				if !ok && flag.Has(flags.AllowJSONStructTags) {
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
				if name != "" || !f.Anonymous || rt.Kind() != reflect.Struct {
					// Normal struct field
					if name == "" {
						name = f.Name
					}
					new := &StructField{
						Name:    name,
						Type:    rt,
						Indices: indices,
					}
					strFields.Flat = append(strFields.Flat, new)
					strFields.Fields[lowerName(name)] = new
					continue
				}
				// Embedded struct
				nextFields = append(nextFields, &StructField{
					Name:    name,
					Type:    rt,
					Indices: indices,
				})
				if flag.Has(flags.KeyedEmbeddedFields) {
					// Add embedded struct to field list
				}
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
