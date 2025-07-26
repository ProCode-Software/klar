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

type FlagDef struct {
	Type        FlagType
	Description string
	Default     any
	Options     any
}

type FlagError struct {
	Message string
	Flag    string
	Arg     string
}

type ArgParser struct {
	NoShift      bool
	AllowUnknown bool
	ExpArgCount  int

	Args  []string
	Flags map[string]any

	defs    map[string]FlagDef
	aliases map[string]string

	IsHelp bool
	Error  *FlagError
}

func NewArgParser(argc int) *ArgParser {
	return &ArgParser{
		Flags:       make(map[string]any),
		defs:        make(map[string]FlagDef),
		aliases:     make(map[string]string),
		ExpArgCount: argc,
	}
}

func (t *ArgParser) BoolFlag(
	name, description string, defaultValue bool, aliases ...string,
) *ArgParser {
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
	var def any
	switch options := options.(type) {
	case map[string]any:
		def = options[defaultValue]
	case []string:
		def = defaultValue
	}
	t.Flags[name] = def
	t.defs[name] = FlagDef{
		Type:        TypeStringFlag,
		Description: description,
		Default:     def,
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
		t.aliases[alias] = flag
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
loop:
	for _, arg := range os.Args[1:] {
		switch {
		case onlyArgs:
			t.Args = append(t.Args, arg)
		case arg == "--help", arg == "-h":
			t.setFlag(arg, true)
			t.IsHelp = true
			break loop
		case arg == "--":
			onlyArgs = true
		case strings.HasPrefix(arg, "-"):
			setArg(cut(arg))
			if _, ok := t.defs[currArg]; !ok {
				t.Error = &FlagError{"Unknown flag", arg, ""}
				return
			}
			t.setFlag(currArg, true)
		case currArg == "":
			t.Args = append(t.Args, arg)
		default:
			switch kind := t.defs[currArg].Type; kind {
			case TypeBoolFlag:
				switch arg {
				case "true":
					t.setFlag(currArg, true)
				case "false":
					t.setFlag(currArg, false)
				default:
					t.Args = append(t.Args, arg)
				}
			case TypeStringFlag:
				t.setFlag(currArg, arg)
			case TypeOptionFlag:
				t.setOpt(currArg, arg)
			default:
				t.Error = &FlagError{"Unknown flag", arg, ""}
				return
			}
			setArg("")
		}
	}
	if t.NoShift && len(t.Args) > 1 {
		t.Args = t.Args[1:]
	} else if len(t.Args) > 2 {
		t.Args = t.Args[2:]
	}
}

func cut(flag string) string {
	return strings.TrimLeft(flag, "-")
}

func (t *ArgParser) setFlag(flag string, value any) {
	if flag == "" {
		return
	}
	f := cut(flag)
	if _, ok := t.Flags[f]; ok {
		t.Error = &FlagError{"Flag already defined", flag, ""}
		return
	}
	t.Flags[f] = value
}

func (t *ArgParser) setOpt(currFlag, arg string) {
	lowerArg := strings.ToLower(arg)
	switch opts := t.defs[currFlag].Options.(type) {
	case map[string]any:
		if val, ok := opts[lowerArg]; ok {
			t.setFlag(currFlag, val)
			return
		}
	case []string:
		for _, opt := range opts {
			if lowerArg == opt {
				t.setFlag(currFlag, opt)
				return
			}
		}
	default:
		panic("invalid type for arg " + currFlag)
	}
	// Invalid value 'arg' for flag '--flag'
	t.Error = &FlagError{"Invalid value", arg, currFlag}
}

func (t *ArgParser) IsDefault(flag string) bool {
	return t.Flags[flag] == t.defs[flag].Default
}
