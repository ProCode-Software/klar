package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

type FlagType int

const (
	_ FlagType = iota
	TypeBoolFlag
	TypeStringFlag
	TypeOptionFlag
)

type CmdInfo struct {
	Name        string
	Description string
	Aliases     []string
}

type FlagDef struct {
	Type        FlagType
	Description string
	Default     any
	Options     any
}

type ArgParser struct {
	NoShift      bool
	AllowUnknown bool
	ExpArgCount  int

	Args  []string
	Flags map[string]any

	defs    map[string]FlagDef
	aliases map[string]string
	Command CmdInfo
}

func NewArgParser(cmd CmdInfo, argc int) *ArgParser {
	return &ArgParser{
		Flags:       make(map[string]any),
		defs:        make(map[string]FlagDef),
		aliases:     make(map[string]string),
		ExpArgCount: argc,
		Command:     cmd,
	}
}

func addPrefix(name string) string {
	if len(name) > 1 {
		return "--" + name
	} else {
		return "-" + name
	}
}

func (t *ArgParser) BoolFlag(
	name, description string, defaultValue bool, aliases ...string,
) *ArgParser {
	name = addPrefix(name)
	t.Flags[name] = defaultValue
	t.defs[name] = FlagDef{
		Type:        TypeBoolFlag,
		Description: description,
		Default:     defaultValue,
	}
	t.alias(name, aliases...)
	return t
}

func (t *ArgParser) StringFlag(
	name, description, defaultValue string, aliases ...string,
) *ArgParser {
	name = addPrefix(name)
	t.Flags[name] = defaultValue
	t.defs[name] = FlagDef{
		Type:        TypeStringFlag,
		Description: description,
		Default:     defaultValue,
	}
	t.alias(name, aliases...)
	return t
}

func (t *ArgParser) OptionFlag(
	name, description string, options any, defaultValue string, aliases ...string,
) *ArgParser {
	name = addPrefix(name)
	switch options := options.(type) {
	case map[string]any:
		t.Flags[name] = options[defaultValue]
	default:
		t.Flags[name] = defaultValue
	}
	t.defs[name] = FlagDef{
		Type:        TypeStringFlag,
		Description: description,
		Default:     defaultValue,
		Options:     options,
	}
	t.alias(name, aliases...)
	return t
}

func (t *ArgParser) ArgAt(i int) string {
	if len(t.Args) > i {
		return t.Args[i]
	}
	return ""
}

func (t *ArgParser) Flag(name string) any {
	return t.Flags[t.resolve(name)]
}

func (t *ArgParser) resolve(flag string) string {
	if alias, ok := t.aliases[flag]; ok {
		return alias
	}
	return flag
}

func (t *ArgParser) alias(flag string, aliases ...string) *ArgParser {
	for _, alias := range aliases {
		t.aliases[addPrefix(alias)] = flag
	}
	return t
}

func (t *ArgParser) Parse() {
	var currArg string
	var onlyArgs bool

	setArg := func(s string) {
		if s == "" {
			currArg = ""
			return
		}
		currArg = t.resolve(s)
	}
	for _, arg := range os.Args[1:] {
		switch {
		case onlyArgs:
			t.Args = append(t.Args, arg)
		case arg == "--help", arg == "-h":
			t.PrintHelp()
			os.Exit(0)
		case arg == "--":
			onlyArgs = true
		case strings.HasPrefix(arg, "-"):
			if currArg != "" {
				t.Flags[currArg] = true
			}
			setArg(arg)
			if _, ok := t.defs[currArg]; !ok {
				InvalidUsage("Unknown flag", arg, t.Usage())
			}
		default:
			switch kind := t.defs[currArg].Type; kind {
			case TypeBoolFlag:
				switch arg {
				case "true":
					t.Flags[currArg] = true
				case "false":
					t.Flags[currArg] = false
				default:
					t.Args = append(t.Args, arg)
				}
			case TypeStringFlag:
				t.Flags[currArg] = arg
			case TypeOptionFlag:
				t.setOpt(currArg, arg)
			default:
				InvalidUsage("Unknown flag", arg, t.Usage())
			}
			setArg("")
		case currArg == "":
			t.Args = append(t.Args, arg)
		}
	}
	if currArg != "" {
		t.Flags[currArg] = true
	}
	if t.NoShift && len(t.Args) > 1 {
		t.Args = t.Args[1:]
	} else if len(t.Args) > 2 {
		t.Args = t.Args[2:]
	}
}

func (t *ArgParser) setOpt(currFlag, arg string) {
	lowerArg := strings.ToLower(arg)
	switch opts := t.defs[currFlag].Options.(type) {
	case map[string]any:
		if val, ok := opts[lowerArg]; ok {
			t.Flags[currFlag] = val
			return
		}
	case []string:
		for _, opt := range opts {
			if lowerArg == opt {
				t.Flags[currFlag] = opt
				return
			}
		}
	default:
		panic("invalid type for arg " + currFlag)
	}
	InvalidUsage(
		fmt.Sprintf("Invalid value %s for flag", ansi.Cyan(arg)),
		currFlag, t.Usage(),
	)
}

func (t ArgParser) Usage() string {
	return ""
}

func (t ArgParser) PrintHelp() {
}
