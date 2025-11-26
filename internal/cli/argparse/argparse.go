package argparse

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
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
	AllowUnknownFlags bool // Whether to allow unknown flags
	ShiftFirst        bool
	InputArgs         []string // The input arguments to parse; default: [os.Args]
	Pattern           []string
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

func first(s string) byte {
	if len(s) == 0 {
		return 0
	}
	return s[0]
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
		ArgDefinitions:  make([]ArgDefinition, len(pattern)),
		ArgNames:        make(map[string]int, len(pattern)),
		FlagDefinitions: make(map[string]FlagDefinition),
	}
	var hasVariadic, hasOptional bool
	for i, pat := range pattern {
		if pre := first(pat); pre != '[' && pre != '<' {
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
	p.Pattern = pattern
	return p
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
//   - If '--help' or '-h' is passed as a flag, Parse immediately returns [HelpError].
//   - If [Parser.AllowUnknownFlags] is false, Parse reports an error when an
//     unknown argument is encountered.
func (p *Parser) Parse() (err error) {
	if p.InputArgs == nil {
		p.InputArgs = os.Args[1:]
	}
	if p.ShiftFirst {
		p.InputArgs = p.InputArgs[1:]
	}
	if len(p.FlagDefinitions) > 0 && p.Flags == nil {
		p.Flags = make(map[string]Flag, len(p.FlagDefinitions)/3)
	}
	for i := 0; i < len(p.InputArgs); i++ {
		item := p.InputArgs[i]
		switch {
		case item == "--":
			p.Args = append(p.Args, p.InputArgs[i+1:]...)
			return
		case item == "--help", item == "-h":
			return &HelpError{}
		case item == "-":
			fallthrough
		default:
			p.Args = append(p.Args, item)
		case strings.HasPrefix(item, "--"):
			// Long flag
			name := p.resolve(item[2:])
			if _, ok := p.FlagDefinitions[name]; !ok {
				if !p.AllowUnknownFlags {
					return &UnknownFlagError{name}
				}
				continue
			}
			var (
				def  = p.FlagDefinitions[name]
				next string
				skip bool
				flag Flag
				n    = i + 1
			)
			isValue := func(flag string) bool { return first(flag) != '-' || flag == "" }
			switch {
			case n < len(p.InputArgs) && isValue(p.InputArgs[i]):
				next = p.InputArgs[i+1]
			case def.Type == TypeBoolFlag:
				flag = newBoolFlag(i, true)
			default:
				return &MissingValueError{name, def.Type}
			}
			switch kind := def.Type; kind {
			case TypeBoolFlag:
				if next == "true" || next == "false" {
					flag = newBoolFlag(i, next == "true")
					skip = true
				}
			case TypeStringFlag:
				flag = &StringFlag{baseFlag: baseFlag{idx: i}, Val: next}
			case TypeNumberFlag:
				num, err := p.checkNumber(name, next)
				if err != nil {
					return err
				}
				flag = &NumberFlag{baseFlag: baseFlag{idx: i}, Val: num}
				skip = true
			case TypeEnumFlag:
				val, err := p.checkEnum(name, next)
				if err != nil {
					return err
				}
				flag = &EnumFlag{baseFlag: baseFlag{idx: i}, Val: val, Name: next}
				skip = true
			case TypeListFlag:
				// Add existing items
				var f *ListFlag
				if existing, ok := p.Flags[name].(*ListFlag); ok {
					f = existing
				} else {
					f = &ListFlag{baseFlag: baseFlag{idx: i}, Val: []any{next}}
				}
				for n := i + 2; n < len(p.InputArgs); n++ {
					item := p.InputArgs[n]
					if len(item) == 0 || item[len(item)-1] != ',' {
						break
					}
					var (
						itemName = item[:len(item)-1]
						val      any
						err      error
					)
					switch def.ItemType {
					case TypeStringFlag:
						f.Val = append(f.Val, itemName)
						continue
					case TypeNumberFlag:
						val, err = p.checkNumber(name, next)
					case TypeEnumFlag:
						val, err = p.checkEnum(name, next)
					default:
						panic("list of list flags are not currently supported")
					}
					if err != nil {
						return err
					}
					f.Val = append(f.Val, val)
				}
				skip = true
				flag = f
			}
			p.Flags[name] = flag
			if skip {
				i++
			}
		case item[0] == '-':
			// Short flag(s)
			val, shouldSkip := true, false
			if nextI := i + 1; nextI < len(p.InputArgs) {
				if next := p.InputArgs[nextI]; next == "true" || next == "false" {
					val = next == "true"
					shouldSkip = true
				}
			}
			for name := range strings.SplitSeq(item[1:], "") {
				name = p.resolve(name)
				def, ok := p.FlagDefinitions[name]
				switch {
				case !ok && !p.AllowUnknownFlags:
					return &UnknownFlagError{name}
				case def.Type != TypeBoolFlag:
					return &InvalidBoolError{name}
				default:
					p.Flags[name] = newBoolFlag(i, val)
				}
			}
			if shouldSkip {
				i++ // Skip the true or false after
			}
		}
	}
	lastArgI, argc := len(p.ArgDefinitions)-1, len(p.Args)
	switch {
	case lastArgI < 0:
		break
	case argc < len(p.ArgDefinitions) && !p.ArgDefinitions[lastArgI].Optional:
		missingLn := len(p.ArgDefinitions) - argc
		missingNames := make([]string, missingLn)
		for i := range missingNames {
			missingNames[i] = p.ArgDefinitions[argc-missingLn+i].Name
		}
		return &MissingArgsError{Missing: missingNames}
	case argc > len(p.ArgDefinitions) && !p.ArgDefinitions[lastArgI].Variadic:
		return &ExtraneousArgsError{p.Args[len(p.ArgDefinitions):]}
	}
	/* // Set defaults
	for name, def := range p.FlagDefinitions {
		if _, ok := p.Flags[name]; !ok {
			p.Flags[name] = def.Default
		}
	} */
	return
}

func newBoolFlag(i int, val bool) Flag {
	return &BoolFlag{baseFlag: baseFlag{idx: i}, Val: val}
}

func (p *Parser) checkEnum(flag, input string) (any, error) {
	if p.enumOpts == nil || p.enumOpts[flag] == nil {
		panic("enum options not set for flag " + flag)
	}
	val, ok := p.enumOpts[flag][input]
	if !ok {
		// TODO: ExpOptions in A-Z order
		return nil, &InvalidOptionError{Flag: flag, ExpOptions: nil}
	}
	return val, nil
}

func (p *Parser) checkNumber(flag, input string) (val float64, err error) {
	val, err = strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, &InvalidNumberError{flag, input}
	}
	return
}

// VarArgByName returns the variadic parameters for the argument named arg.
// It panics if arg is not defined or is not the last argument.
func (p *Parser) VarArgByName(arg string) []string {
	if len(p.ArgDefinitions) == 0 {
		panic("argument " + arg + " not defined")
	} else if len(p.Args) == 0 {
		return p.Args[:0]
	}
	// Variadic argument is guaranteed to be the last argument
	defsBefore := len(p.ArgDefinitions) - 1
	lastDef := p.ArgDefinitions[defsBefore]
	if lastDef.Name != arg {
		panic("argument " + arg + " not defined or is not the last argument")
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
	if i >= len(p.Args) {
		return ""
	}
	return p.Args[i]
}

// Flag returns the value of the flag name.
func (p *Parser) Flag(name string) Flag {
	name = p.resolve(name)
	f, ok := p.Flags[name]
	if !ok {
		return p.FlagDefinitions[name].Default
	}
	return f
}

func (p *Parser) resolve(name string) string {
	if target, ok := p.FlagAliases[name]; ok {
		return target
	}
	return name
}

func FormatFlag(flag string) string {
	if len(flag) == 1 {
		return "-" + flag
	}
	return "--" + flag
}

// cutDashes removes leading dashes from s.
func cutDashes(s string) string {
	return strings.TrimLeft(s, "-")
}

func (p *Parser) BoolFlag(name, desc string, def bool, aliases ...string) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeBoolFlag,
		Default:     &BoolFlag{Val: def},
		Description: desc,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) StringFlag(name, desc, param, def string, aliases ...string) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeStringFlag,
		Default:     &StringFlag{Val: def},
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) OptionFlag(
	name, desc, param string,
	opts map[string]any, def string, aliases ...string,
) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeEnumFlag,
		Default:     &EnumFlag{Val: opts[def], Name: def},
		Description: desc,
		ParamName:   param,
	}
	if p.enumOpts == nil {
		p.enumOpts = map[string]map[string]any{name: opts}
	} else {
		p.enumOpts[name] = opts
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) NumberFlag(name, desc, param string, def float64, aliases ...string) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeNumberFlag,
		Default:     &NumberFlag{Val: def},
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) ListFlag(name, desc, param string, def []any, aliases ...string) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeListFlag,
		Default:     &ListFlag{Val: def},
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	return p
}

func (p *Parser) ListFlagOf(
	name, desc, param string,
	itemType FlagType, opts map[string]any,
	def []any, aliases ...string,
) *Parser {
	p.FlagDefinitions[name] = FlagDefinition{
		Type:        TypeListFlag,
		ItemType:    itemType,
		Default:     &ListFlag{Val: def},
		Description: desc,
		ParamName:   param,
	}
	p.makeAliases(name, aliases)
	if opts == nil {
		return p
	}
	if p.enumOpts == nil {
		p.enumOpts = map[string]map[string]any{name: opts}
	} else {
		p.enumOpts[name] = opts
	}
	return p
}

func (p *Parser) makeAliases(flag string, aliases []string) {
	if p.FlagAliases == nil {
		p.FlagAliases = make(map[string]string, len(aliases)+1)
	}
	for _, alias := range aliases {
		p.FlagAliases[alias] = flag
	}
}

// ResolveFlag finds the flag name from any alises. ResolveFlag returns name if not found.
func (p *Parser) ResolveFlag(name string) string {
	return p.resolve(name)
}
