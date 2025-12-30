package argparse

import (
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// TODO: Parser.SetOptions(flag string, optMap map[passed string]any)
// and Parser.SetDefaults
type Parser struct {
	AllowUnknownFlags bool     // Whether to allow unknown flags
	StartOffset       int      // Number of arguments at the beginning to shift
	InputArgs         []string // The input arguments to parse; default: [os.Args]
	Pattern           []string // The pattern as provided to [NewParser]
	FlagDefs          map[string]FlagDef
	ArgDefs           []ArgDef
	ArgNames          map[string]int
	FlagAliases       map[string]string

	Args  []string         // Parsed arguments only
	Flags map[string]*Flag // Parsed flags only

	reflector *reflector // When [FromStruct] is used
	enumOpts  map[string]map[string]any
}

type FlagDef struct {
	Type        FlagType
	Default     *Flag
	Description string
	ParamName   string   // Name of flag parameter: --flag <param>
	ItemType    FlagType // For [TypeListFlag]
}

type ArgDef struct {
	Name               string
	Optional, Variadic bool
}

type reflector struct {
	args, flags map[string]reflect.Value
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
	p := &Parser{
		ArgDefs:  make([]ArgDef, len(pattern)),
		ArgNames: make(map[string]int, len(pattern)),
		FlagDefs: make(map[string]FlagDef),
	}
	var hasVariadic, hasOptional bool
	for i, pat := range pattern {
		var optional bool
		switch {
		case len(pat) < 3:
			fallthrough
		default:
			panic(fmt.Sprintf("invalid pattern %s (#%d)", pat, i))
		case pat[0] == '<' && pat[len(pat)-1] == '>':
		case pat[0] == '[' && pat[len(pat)-1] == ']':
			optional = true
		}
		name, variadic := strings.CutSuffix(pat[1:len(pat)-1], "...")
		// There can only be 1 optional argument
		if optional {
			hasOptional = true
		} else if hasOptional {
			errReqBeforeOpt(name, i+1)
		}
		// Same for variadic arguments
		if hasVariadic {
			errVariadicLast(name, i+1)
		} else if variadic {
			hasVariadic = true
		}
		p.ArgDefs[i] = ArgDef{
			Name:     name,
			Optional: optional,
			Variadic: variadic,
		}
		p.ArgNames[name] = i
	}
	p.Pattern = pattern
	return p
}

func (d ArgDef) String() string {
	var variadic string
	if d.Variadic {
		variadic = "..."
	}
	if d.Optional {
		return "[" + d.Name + variadic + "]"
	}
	return "<" + d.Name + variadic + ">"
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

// Parse parses flags and arguments from [Parser.InputArgs] (or [os.Args] if nil),
// into p.Flags and p.Args.
//
//   - Items starting with '-' are treated as flags.
//   - Short flags are defined with a single '-'. Long flags are defined with '--'.
//     A single '-' followed by multiple character is treated as multiple boolean flags.
//   - Missing arguments or flag values return an error.
//   - If '--' (as a whole flag) is passed, it is skipped and remaining items are
//     parsed as arguments, even if they start with '-'.
//   - If '--help' or '-h' is passed as a flag, Parse immediately returns [HelpError].
//   - If [Parser.AllowUnknownFlags] is false, Parse reports an error when an
//     unknown argument is encountered.
func (p *Parser) Parse() (err error) {
	if p.InputArgs == nil {
		p.InputArgs = os.Args[1:]
	}
	// Shift the input arguments
	if p.StartOffset < len(p.InputArgs) {
		p.InputArgs = p.InputArgs[p.StartOffset:]
	} else {
		// Most likely not enough arguments
		p.InputArgs = p.InputArgs[:0]
	}
	if len(p.FlagDefs) > 0 && p.Flags == nil {
		p.Flags = make(map[string]*Flag, len(p.FlagDefs)/3)
	}
loop:
	for i := 0; i < len(p.InputArgs); i++ {
		item := p.InputArgs[i]
		switch {
		case item == "--":
			p.Args = append(p.Args, p.InputArgs[i+1:]...)
			break loop
		case item == "--help", item == "-h":
			return HelpError{}
		case item == "-":
			// Usually means "read from stdin". Treat as an argument
			fallthrough
		default:
			p.Args = append(p.Args, item)
		case strings.HasPrefix(item, "--"):
			// Long flag
			if i, err = p.parseLongFlag(i, item); err != nil {
				return err
			}
		case item[0] == '-':
			// Short flag(s)
			if i, err = p.parseShortFlag(i, item); err != nil {
				return err
			}
		}
	}
	last, n := len(p.ArgDefs)-1, len(p.Args)
	switch {
	case last < 0:
	case n < len(p.ArgDefs) && !p.ArgDefs[last].Optional:
		// Not enough arguments
		var b strings.Builder
		for i := n; i < len(p.ArgDefs); i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(p.ArgDefs[i].String())
		}
		return &MissingArgsError{b.String()}
	case n > len(p.ArgDefs) && (len(p.ArgDefs) == 0 || !p.ArgDefs[last].Variadic):
		// Too many arguments
		return &ExtraArgsError{p.Args[len(p.ArgDefs):]}
	}
	return nil
}

func (p *Parser) parseShortFlag(i int, item string) (int, error) {
	val, skip := true, false
	if len(item) == 2 && i+1 < len(p.InputArgs) {
		name := item[1:]
		def, ok := p.FlagDefs[name]
		if !ok {
			if !p.AllowUnknownFlags {
				return i, &UnknownFlagError{name}
			}
			return i, nil
		}
		flag, i, err := p.parseValue(&def, name, p.InputArgs[i+1], i)
		p.Flags[name] = flag
		return i, err
	}
	if nextI := i + 1; nextI < len(p.InputArgs) {
		if next := p.InputArgs[nextI]; next == "true" || next == "false" {
			val = next == "true"
			skip = true
		}
	}
	for name := range strings.SplitSeq(item[1:], "") {
		name = p.resolve(name)
		def, ok := p.FlagDefs[name]
		switch {
		case !ok && !p.AllowUnknownFlags:
			return i, &UnknownFlagError{name}
		case def.Type != TypeBool:
			return i, &MissingValueError{name, def.Type}
		default:
			p.Flags[name] = newDeclaredFlag(TypeBool, i, val)
		}
	}
	if skip {
		i++ // Skip the true or false after
	}
	return i, nil
}

func (p *Parser) parseLongFlag(i int, item string) (j int, err error) {
	name := p.resolve(item[2:])
	def, ok := p.FlagDefs[name]
	switch {
	case !ok:
		if !p.AllowUnknownFlags {
			return i, &UnknownFlagError{name}
		}
		return i, nil
	case p.Flags[name] != nil && def.Type != TypeList:
		return i, &RepeatedFlagError{name}
	case i+1 < len(p.InputArgs) && (p.InputArgs[i+1] == "" || p.InputArgs[i+1][0] != '-'):
		var flag *Flag
		flag, i, err = p.parseValue(&def, name, p.InputArgs[i+1], i)
		p.Flags[name] = flag
		return i, err
	case def.Type == TypeBool:
		p.Flags[name] = newDeclaredFlag(TypeBool, i, true)
		return i, nil
	default:
		return i, &MissingValueError{name, def.Type}
	}
}

func (p *Parser) parseValue(
	def *FlagDef, name, next string, i int,
) (flag *Flag, j int, err error) {
	switch kind := def.Type; kind {
	case TypeBool:
		if next == "true" || next == "false" {
			return newDeclaredFlag(TypeBool, i, next == "true"), i + 1, nil
		}
		return nil, i, &InvalidValueError{TypeBool, name, next}
	case TypeString:
		return newDeclaredFlag(TypeString, i, next), i + 1, nil
	case TypeInt:
		num, err := p.checkInt(name, next)
		return newDeclaredFlag(TypeInt, i, num), i + 1, err
	case TypeFloat:
		num, err := p.checkFloat(name, next)
		return newDeclaredFlag(TypeFloat, i, num), i + 1, err
	case TypeEnum:
		val, err := p.checkEnum(name, next)
		return newDeclaredFlag(TypeEnum, i, &Enum{next, val}), i + 1, err
	case TypeList:
		items, err := p.parseList(name, def, next)
		return newDeclaredFlag(TypeList, i, items), i + 1, err
	default:
		panic("unreachable")
	}
}

func (p *Parser) parseList(name string, def *FlagDef, next string) (items any, err error) {
	// Add existing items
	if existing, ok := p.Flags[name]; ok {
		items = existing
	}
	for item := range strings.SplitSeq(next, ",") {
		if item == "" {
			continue
		}
		switch def.ItemType {
		case TypeString:
			items, err = addItem(item, items, func(_, s string) (string, error) {
				return s, nil
			})
		case TypeInt:
			items, err = addItem(item, items, p.checkInt)
		case TypeFloat:
			items, err = addItem(item, items, p.checkFloat)
		case TypeBool:
			items, err = addItem(item, items, func(_, b string) (bool, error) {
				if b == "true" || b == "false" {
					return b == "true", nil
				}
				return false, &InvalidValueError{TypeBool, name, b}
			})
		case TypeEnum:
			// Use the flag's name for the enum options
			items, err = addItem(item, items, func(_, o string) (any, error) {
				return p.checkEnum(name, o)
			})
		default:
			panic("list of list flags are not currently supported")
		}
		if err != nil {
			return items, err
		}
	}
	return items, nil
}

func addItem[T any](str string, list any, check func(a, b string) (T, error)) (any, error) {
	v, err := check("", str)
	if err != nil {
		return list, err
	}
	// Append the item
	if list == nil {
		return []T{v}, nil
	}
	return append(list.([]T), v), nil
}

func newDeclaredFlag(kind FlagType, i int, v any) *Flag {
	return &Flag{Type: kind, Value: v, Index: i, Set: true}
}

func (p *Parser) resolve(name string) string {
	if target, ok := p.FlagAliases[name]; ok {
		return target
	}
	return name
}

func (p *Parser) makeAliases(flag string, aliases []string) {
	if p.FlagAliases == nil {
		p.FlagAliases = make(map[string]string, len(aliases)+1)
	}
	for _, alias := range aliases {
		p.FlagAliases[alias] = flag
	}
}

// Types
// ========

func (p *Parser) checkEnum(flag, input string) (any, error) {
	if p.enumOpts == nil || p.enumOpts[flag] == nil {
		panic("enum options not set for flag " + flag)
	}
	val, ok := p.enumOpts[flag][input]
	if !ok {
		return nil, &InvalidOptionError{
			Flag:       flag,
			ExpOptions: slices.Sorted(maps.Keys(p.enumOpts[flag])),
			Input:      input,
		}
	}
	return val, nil
}

func (p *Parser) checkFloat(flag, input string) (float64, error) {
	val, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, &InvalidValueError{TypeFloat, flag, input}
	}
	return val, nil
}

func (p *Parser) checkInt(flag, input string) (int, error) {
	val, err := strconv.ParseInt(input, 0, 64)
	if err != nil {
		return 0, &InvalidValueError{TypeInt, flag, input}
	}
	return int(val), nil
}
