package cli

import (
	"os"
	"strings"
)

type FlagType int

const (
	TypeBoolFlag FlagType = iota
	TypeStringFlag
)

type FlagDefinition struct {
	Type        FlagType
	Description string
	Default     any
}

type ArgTable struct {
	NoShift     bool
	Args        []string
	Flags       map[string]any
	Definitions map[string]FlagDefinition
}

func addPrefix(name string) string {
	if len(name) > 1 {
		return "--" + name
	} else {
		return "-" + name
	}
}

func (t *ArgTable) BoolFlag(name, description string, defaultValue bool) {
	name = addPrefix(name)
	t.Definitions[name] = FlagDefinition{TypeBoolFlag, description, defaultValue}
}

func (t *ArgTable) StringFlag(name, description, defaultValue string) {
	name = addPrefix(name)
	t.Definitions[name] = FlagDefinition{TypeStringFlag, description, defaultValue}
}

func (t ArgTable) ArgAt(i int) string {
	if len(t.Args) > i {
		return t.Args[i]
	}
	return ""
}

func (t *ArgTable) Parse() {
	currArg := ""
	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, "-"):
			if currArg != "" {
				t.Flags[currArg] = true
			}
			currArg = arg
		case currArg != "":
			if t.Definitions[arg].Type == TypeBoolFlag {
				if arg == "true" {
					t.Flags[currArg] = true
				} else if arg == "false" {
					t.Flags[currArg] = false
				}
			} else {
				t.Flags[currArg] = arg
			}
			currArg = ""
		default:
			t.Args = append(t.Args, arg)
		}
	}
	if !t.NoShift {
		t.Args = t.Args[1:]
	}
}

func (t ArgTable) PrintHelp(cmdName string) {
}
