package cli

import (
	"os"
	"strings"
)

type FlagType int

const (
	_ FlagType = iota
	TypeBoolFlag
	TypeStringFlag
	TypeOptionFlag
)

type CommandInfo struct {
	Name        string
	Description string
	Aliases     []string
}

type FlagDef struct {
	Type        FlagType
	Description string
	Default     any
	Options     []string
}

type ArgParser struct {
	NoShift      bool
	AllowUnknown bool

	Args  []string
	Flags map[string]any

	defs    map[string]FlagDef
	aliases map[string]string
}

func NewArgParser() *ArgParser {
	return &ArgParser{
		Flags:   make(map[string]any),
		defs:    make(map[string]FlagDef),
		aliases: make(map[string]string),
	}
}

func addPrefix(name string) string {
	if len(name) > 1 {
		return "--" + name
	} else {
		return "-" + name
	}
}

func (t *ArgParser) BoolFlag(name, description string, defaultValue bool) {
	name = addPrefix(name)
	t.defs[name] = FlagDef{
		Type:        TypeBoolFlag,
		Description: description,
		Default:     defaultValue,
	}
}

func (t *ArgParser) StringFlag(name, description, defaultValue string) {
	name = addPrefix(name)
	t.defs[name] = FlagDef{
		Type:        TypeStringFlag,
		Description: description,
		Default:     defaultValue,
	}
}

func (t *ArgParser) OptionFlag(
	name, description string, options []string, defaultValue string,
) {
	name = addPrefix(name)
	t.defs[name] = FlagDef{
		Type:        TypeStringFlag,
		Description: description,
		Default:     defaultValue,
		Options:     options,
	}
}

func (t *ArgParser) ArgAt(i int) string {
	if len(t.Args) > i {
		return t.Args[i]
	}
	return ""
}

func (t *ArgParser) Alias(flag string, aliases ...string) {
	flag = addPrefix(flag)
	for _, alias := range aliases {
		t.aliases[addPrefix(alias)] = flag
	}
}

func (t *ArgParser) Parse() {
	var currArg string
	var onlyArgs bool

loop:
	for _, arg := range os.Args[1:] {
		switch {
		case onlyArgs:
			t.Args = append(t.Args, arg)
		case arg == "--":
			break loop
		case strings.HasPrefix(arg, "-"):
			if currArg != "" {
				t.Flags[currArg] = true
			}
			currArg = arg
		default:
			switch kind := t.defs[arg].Type; kind {
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
				lowerArg := strings.ToLower(arg)
				for _, opt := range t.defs[currArg].Options {
					if lowerArg == opt {
						t.Flags[currArg] = opt
						break
					}
				}
				Failure("Unknown option '%s' for flag '%s'", arg, currArg)
			default:
				Failure("Unknown flag '%s'", arg)
			}
			currArg = ""
		case currArg == "":
			t.Args = append(t.Args, arg)
		}
	}
	if t.NoShift && len(t.Args) > 1 {
		t.Args = t.Args[1:]
	} else if len(t.Args) > 2 {
		t.Args = t.Args[2:]
	}
}

func (t ArgParser) PrintHelp(cmd CommandInfo) {
}
