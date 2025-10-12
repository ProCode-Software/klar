package argparse

import (
	"fmt"
	"os"
	"reflect"
	"strings"
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
	Index() int
}

type FlagDefinition struct {
	Type        FlagType
	Default     Flag
	Description string
	ParamName   string   // Name of flag parameter: --flag <param>
	ItemType    FlagType // For [TypeListFlag]
}

type ArgDefinition struct {
	Name               string
	Optional, Variadic bool
}

// TODO: Parser.SetOptions(flag string, optMap map[passed string]any)
// and Parser.SetDefaults
type Parser struct {
	AllowUnknownFlags bool     // Whether to allow unknown flags
	InputArgs         []string // The input arguments to parse; default: [os.Args]
	FlagDefinitions   map[string]FlagDefinition
	ArgDefinitions    []ArgDefinition
	ArgNames          map[string]int
	FlagAliases       map[string]string
	MinArgs, MaxArgs  int

	Args  []string        // Parsed arguments only
	Flags map[string]Flag // Parsed flags only

	argReflector, flagReflector map[string]reflect.Value // When [FromStruct] is used
	enumOpts                    map[string]map[string]any
}

// NewParser returns a [*Parser] with arguments specified in pattern.
//
// The syntax for each item in pattern is:
//
//	[arg]     Optional argument.
//	<arg>     Mandatory argument. Must go before any optional arguments
//	<arg...>  Variadic argument requiring at least 1 parameter. Must be the last argument.
//	[arg...]  Variadic argument accepting any amount of parameters. Must be the last argment.
func NewParser(pattern ...string) *Parser {
	p := &Parser{ArgDefinitions: make([]ArgDefinition, len(pattern))}
	var hasVariadic, hasOptional bool
	for i, pat := range pattern {
		// Also checks if empty
		if pre := pat[:0]; pre != "[" && pre != "<" {
			panic(fmt.Sprintf("invalid pattern %s (#%d)", pat, i))
		}
		optional := pat[0] == '['
		name, variadic := strings.CutSuffix(pat[1:len(pat)-1], "...")
		if optional {
			hasOptional = true
		} else if hasOptional {
			errReqBeforeOpt(name, i+1)
		}
		if hasVariadic {
			errVariadicLast(name, i+1)
		} else if variadic {
			hasVariadic = true
		}
		p.ArgDefinitions[i] = ArgDefinition{
			Name:     name,
			Optional: optional,
			Variadic: variadic,
		}
		p.ArgNames[name] = i
	}
	return p
}

// cutDashes removes leading dashes from s.
func cutDashes(s string) string {
	return strings.TrimLeft(s, "-")
}

// errReqBeforeOpt panics if an optional argument has already been declared
// (hasOptional == true) and argument f is not optional (optional == false).
func errReqBeforeOpt(f string, i int) {
	panic(fmt.Sprintf(
		"non-optional argument %s (#%d) must go before other optional arguments",
		f, i,
	))
}

// errVariadicLast panics if hasVariadic == true.
func errVariadicLast(f string, i int) {
	panic(fmt.Sprintf("can't declare argument %s (#%d) after variadic parameter", f, i))
}

// TODO: finish
// Parse parses flags and arguments from [Parser.InputArgs] (or [os.Args] if nil),
// into p.Flags and p.Args.
//
//   - Items starting with '-' are treated as flags.
//   - Short flags are defined with a single '-'. Long flags are defined with '--'.
//     A single '-' followed by multiple character is treated as multiple boolean flags.
//   - Missing arguments or flag values return an error.
//   - If '--' (as a whole flag) is passed, it is skipped and remaining items are
//     parsed as arguments, even if they start with '-'.
//   - If '--help' or '-h' is passed as a flag, Parse immediately returns [ErrHelp].
//   - If [Parser.AllowUnknownFlags] is false, Parse reports an error when an
//     unknown argument is encountered.
func (p *Parser) Parse() (err error) {
	if p.InputArgs == nil {
		p.InputArgs = os.Args[1:]
	}
	for i, item := range p.InputArgs {
		switch {
		case item == "--":
			p.Args = append(p.Args, p.InputArgs[i+1:]...)
			return
		case item == "--help", item == "-h":
			return ErrHelp
		case item == "-":
			fallthrough
		default:
			p.Args = append(p.Args, item)
		case strings.HasPrefix(item, "--"):
			// Long flag
		case item[0] == '-':
			// Short flag(s)
		}
	}
	return
}

// VariadicArgByName returns the variadic parameters for the argument named arg.
// It panics if arg is not defined or is not the last argument.
func (p *Parser) VariadicArgByName(arg string) []string {
	if len(p.ArgDefinitions) == 0 {
		panic("argument " + arg + " not defined")
	} else if len(p.Args) == 0 {
		return p.Args[:0]
	}
	// Variadic argument is guaranteed to be the last argument
	defsBefore := len(p.ArgDefinitions) - 1
	lastDef := p.ArgDefinitions[defsBefore]
	if lastDef.Name != arg {
		panic("argument " + arg + " not defined")
	}
	return p.Args[defsBefore:]
}

// ArgAt returns the i'th argument in p.Args. The first argument is at index 1.
// If i is out of range, ArgAt returns "". ArgAt panics if i < 1.
func (p *Parser) ArgAt(i int) string {
	switch {
	case i == 0:
		panic("Parser.Arg(0): the first argument index is 1")
	case i < 0:
		panic("negative argument index")
	case i > len(p.Args):
		return ""
	}
	return p.Args[i-1]
}

// ArgByName returns the value of the argument named name. ArgByName panics
// if name is not defined. If name is a variadic parameter, ArgByName returns
// the first value; [Parser.VariadicArgByName] should be used instead.
func (p *Parser) ArgByName(name string) string {
	i, ok := p.ArgNames[name]
	if !ok {
		panic("argument " + name + " not defined")
	}
	return p.Args[i]
}

// Flag returns the value of the flag name. It panics if name is not defined.
func (p *Parser) Flag(name string) Flag {
	f, ok := p.Flags[p.resolve(name)]
	if !ok {
		panic("flag " + name + " not defined")
	}
	return f
}

func (p *Parser) resolve(name string) string {
	if target, ok := p.FlagAliases[name]; ok {
		return target
	}
	return name
}
