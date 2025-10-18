package argparse

import (
	"fmt"
	"reflect"
	"strings"
)

// FromStruct returns a [*Parser] with flags and arguments defined by struct tags
// in the struct v points to. It panics if v is not a pointer to a struct.
//
// Flags and arguments are defined by the arg: struct tag. Values are either
// comma-separated flags starting with '-' (e.g. `arg:"--flag,-f"`), or
// arguments as defined by [NewParser].
func FromStruct[T any](v *T) *Parser {
	var (
		p = &Parser{
			argReflector:  map[string]reflect.Value{},
			flagReflector: map[string]reflect.Value{},
			enumOpts:      map[string]map[string]any{},
		}
		rv = reflect.ValueOf(v).Elem() // Because v is a pointer
		rt = rv.Type()
	)
	if rt.Kind() != reflect.Struct {
		panic("v must be a pointer to a struct")
	}
	var hasOptional, hasVariadic bool
	for i := range rt.NumField() {
		f := rt.Field(i)
		tag := f.Tag.Get("arg")
		if tag == "" || tag == "-" || !f.IsExported() {
			continue
		}
		fieldVal := rv.Field(i)
		switch tag[0] {
		case '-':
			// Flag
			var (
				aliases  = strings.Split(tag, ",")
				flagName = cutDashes(aliases[0])
				def      = defFromReflectValue(f.Type, fieldVal) // Create the flag definition
			)
			p.flagReflector[flagName] = fieldVal
			// The rest are aliases
			for _, alias := range aliases[1:] {
				p.FlagAliases[cutDashes(alias)] = flagName
			}
			p.FlagDefinitions[flagName] = def
		case '[', '<':
			// Argument
			var (
				arg, variadic = strings.CutSuffix(tag[1:len(tag)-1], "...")
				optional      = tag[0] == '['
				pos           = len(p.ArgDefinitions) + 1
			)
			if optional {
				hasOptional = true
			} else if hasOptional {
				errReqBeforeOpt(arg, pos)
			}
			if hasVariadic {
				errVariadicLast(arg, pos)
			} else if variadic {
				hasVariadic = true
			}
			p.ArgDefinitions = append(p.ArgDefinitions, ArgDefinition{
				Name:     arg,
				Optional: optional,
				Variadic: variadic,
			})
			p.ArgNames[arg] = len(p.ArgDefinitions) - 1
			p.argReflector[arg] = fieldVal
		default:
			panic(fmt.Sprintf("invalid arg struct tag for field %s: %q", f.Name, tag))
		}
	}
	return p
}

func typeFromReflectType(kind reflect.Kind) FlagType {
	switch kind {
	case reflect.Bool:
		return TypeBoolFlag
	case reflect.String:
		return TypeStringFlag
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Uintptr, reflect.Float32, reflect.Float64:
		return TypeNumberFlag
	case reflect.Array, reflect.Slice:
		return TypeListFlag
	}
	return TypeEnumFlag
}

func defFromReflectValue(rt reflect.Type, rv reflect.Value) (def FlagDefinition) {
	def.Type = typeFromReflectType(rt.Kind())
	switch def.Type {
	case TypeEnumFlag:
		def.Default = &EnumFlag{Val: rv.Interface()}
	case TypeListFlag:
		def.ItemType = typeFromReflectType(rt.Elem().Kind())
		def.Default = &ListFlag{Val: rv.Interface().([]any)}
	case TypeBoolFlag:
		def.Default = &BoolFlag{Val: rv.Bool()}
	case TypeStringFlag:
		def.Default = &StringFlag{Val: rv.String()}
	case TypeNumberFlag:
		f := &NumberFlag{}
		switch {
		case rv.CanInt():
			f.Val = float64(rv.Int())
		case rv.CanFloat():
			f.Val = rv.Float()
		case rv.CanUint():
			f.Val = float64(rv.Uint())
		}
		def.Default = f
	}
	return
}

func (p *Parser) setStructFields() {
}
