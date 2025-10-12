package argparse

import (
	_ "os"
	"reflect"
)

type FlagType int

const (
	_ FlagType = iota
	TypeBoolFlag
	TypeStringFlag
	TypeEnumFlag
	TypeListFlag
	TypeNumberFlag
)

type Flag interface {
	Type() FlagType
	Value() any
	Name() string
	Index() int
}

type FlagDefinition struct {
	Type    FlagType
	Default Flag

	ItemType FlagType       // For [TypeListFlag]
	Options  map[string]any // For [TypeEnumFlag] and [TypeListFlag]
}

type ArgDefinition struct {
	Optional, Variadic bool
}

// TODO: Parser.SetOptions(flag string, optMap map[passed string]any)
type Parser struct {
	AllowUnknownFlags bool     // Whether to allow unknown flags
	ArgPattern        []string // The pattern of the arguments
	InputArgs         []string // The input arguments to parse; default: [os.Args]
	FlagDefinitions   map[string]FlagDefinition
	ArgDefinitions    map[string]ArgDefinition
	FlagAliases       map[string]string

	Help  bool            // Whether '--help'/'-h' flag was passed
	Args  []string        // Parsed arguments only
	Flags map[string]Flag // Parsed flags only

	argReflector, flagReflector map[string]reflect.Value // When [FromStruct] is used
}
