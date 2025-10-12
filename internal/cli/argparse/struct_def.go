package argparse

import (
	"fmt"
	"reflect"
	"strings"
)

func FromStruct[T any](v *T) *Parser {
	p := &Parser{
		argReflector:  map[string]reflect.Value{},
		flagReflector: map[string]reflect.Value{},
	}
	rv := reflect.ValueOf(v).Elem() // Because v is a pointer
	rt := rv.Type()
	if rt.Kind() != reflect.Struct {
		panic("v must be a pointer to a struct")
	}
	for i := range rt.NumField() {
		field := rt.Field(i)
		tag := field.Tag.Get("arg")
		if tag == "" || tag == "-" || !field.IsExported() {
			continue
		}
		switch tag[0] {
		case '-': // Flag
			allFlags := strings.Split(tag, ",")
			flagName := strings.TrimLeft(allFlags[0], "-")
			p.FlagDefinitions[flagName] = defFromReflectValue(field.Type, rv.Field(i))
			// The rest are aliases
			for _, alias := range allFlags[1:] {
				p.FlagAliases[strings.TrimLeft(alias, "-")] = flagName
			}
		case '[', '<':
		default:
			panic(fmt.Sprintf("invalid arg struct tag for field %s: %q", field.Name, tag))
		}
	}
	return p
}

func defFromReflectValue(rt reflect.Type, rv reflect.Value) (def FlagDefinition) {
	switch rt.Kind() {
	case reflect.Bool:
		def.Type = TypeBoolFlag
	case reflect.String:
		def.Type = TypeStringFlag
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Uintptr, reflect.Float32, reflect.Float64:
		def.Type = TypeNumberFlag
	case reflect.Array, reflect.Slice:
		def.Type = TypeListFlag
	default:
		def.Type = TypeEnumFlag

	}
	return FlagDefinition{}
}
