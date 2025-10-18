package command

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
)

var (
	ExecName string = "klar"
	Commands map[string]*Command
)

type ExampleCmd struct {
	Command     string
	Args        []string
	Flags       []string
	Description string
}

type Command struct {
	Name             string
	ShortDescription string
	Usage            []string
	Aliases          []string
	Run              RunFunc

	// Shown in command help
	Subcommands     []*Command
	Flags           *argparse.Parser
	LongDescription string
	SeeAlso         []string
	Examples        []ExampleCmd
}

type RunFunc func(r *Runner)

type Runner struct {
	Parser *argparse.Parser
}

func (r *Runner) Arg(i int) string                   { return r.Parser.ArgAt(i) }
func (r *Runner) Flag(n string) any                  { return r.Parser.Flag(n) }
func (r *Runner) AllFlags() map[string]argparse.Flag { return r.Parser.Flags }
func (r *Runner) AllArgs() []string                  { return r.Parser.Args }
func (r *Runner) StringFlag(n string) string {
	return r.Flag(n).(*argparse.StringFlag).Val
}

func (r *Runner) BoolFlag(n string) bool {
	return r.Flag(n).(*argparse.BoolFlag).Val
}

func Lookup(
	name string, commands map[string]*Command, aliases map[string]string,
) *Command {
	if cmd, ok := commands[name]; ok {
		return cmd
	}
	if aliases == nil {
		return nil
	}
	return commands[aliases[name]]
}

func Run(cmd *Command) {
	if cmd == nil {
		panic("command: Run(nil)")
	}
	if cmd.Run == nil {
		panic("cannot run command '" + cmd.Name + ": Run function is not defined")
	}
	if cmd.Flags == nil {
		cmd.Flags = argparse.NewParser()
	}
	cmd.Flags.ShiftFirst = true
	cmd.Flags.InputArgs = os.Args[1:]
	if err := cmd.Flags.Parse(); err != nil {
		cmd.handleFlagError(err)
	}
	cmd.Run(&Runner{Parser: cmd.Flags})
}

func (c *Command) handleFlagError(err error) {
	stdout, stderr := os.Stdout, os.Stderr
	_ = stderr
	switch err.(type) {
	case *argparse.ErrHelp:
		c.Help(stdout)
		os.Exit(0)
	case *argparse.ErrInvalidBool:
	case *argparse.ErrExtraneousArgs:
	case *argparse.ErrInvalidNumber:
	case *argparse.ErrInvalidOption:
	case *argparse.ErrMissingArgs:
	case *argparse.ErrMissingValue:
	case *argparse.ErrUnknownFlag:
	}
}

func (c *Command) Help(f io.Writer) {
	if c.Flags != nil {
		c.Usage = c.Flags.Pattern
	}
	execTemplate(newTemplate("help", fullHelpTemplate), f, c)
}

func (c *Command) ArgUsage() string {
	var w strings.Builder
	execTemplate(newTemplate("usage", usageTemplate), &w, c)
	return w.String()
}

func (c *Command) SeeAlsoString(indent int) string {
	b := &strings.Builder{}
	tw := cli.NewTabWriterOutput(b)
	tw.Margin = indent
	for _, cmd := range c.SeeAlso {
		tw.WriteCells(ansi.BoldGreen(ExecName)+" "+ansi.BoldYellow(cmd), getDesc(cmd))
	}
	tw.Flush()
	return b.String()
}

func (c *Command) FlagString(indent int) string {
	if c.Flags == nil {
		return ""
	}
	var (
		b       = &strings.Builder{}
		b2   = strings.Builder{}
		tw      = cli.NewTabWriterOutput(b)
		aliases = make(map[string][]string, len(c.Flags.FlagDefinitions))
		sep     = ansi.Reset() + ", " + ansi.Partial(ansi.CodeCyan)
	)
	tw.Margin = indent
	for alias, actual := range c.Flags.FlagAliases {
		aliases[actual] = append(aliases[actual], alias)
	}
	for name, flag := range c.Flags.FlagDefinitions {
		al := aliases[name]
		b2.Reset()
		sortAliases(al) // Sort aliases by length
		b2.WriteString(ansi.Partial(ansi.CodeCyan))
		if len(al) > 0 && len(al[0]) == 1 {
			// Short alias
			b2.WriteString(
				argparse.FormatFlag(al[0]) + sep + argparse.FormatFlag(name),
			)
			al = al[1:]
		} else {
			b2.WriteString("    " + argparse.FormatFlag(name))
		}
		for _, alias := range al {
			b2.WriteString(sep)
			b2.WriteString(argparse.FormatFlag(alias))
		}
		b2.WriteString(ansi.Reset())
		b2.WriteByte('\t')
		b2.WriteString(flag.Description)
		if flag.Default != nil {
			switch flag.Default.Value() {
			case "", false, 0, nil:
			default:
				b2.WriteString(ansi.Gray(fmt.Sprintf(
					" (default: %v)", flag.Default.Value(),
				)))
			}
		}
		b2.WriteByte('\n')
		tw.WriteString(b2.String())
	}
	tw.Flush()
	return b.String()
}

var templFuncs = template.FuncMap{
	"join":      strings.Join,
	"hasPrefix": strings.HasPrefix,

	"exec":  func() string { return ansi.BoldGreen(ExecName) },
	"ansi":  func(color, s string) string { return ansi.Color("\x1b["+color+"m", s) },
	"bold":  func(color, s string) string { return ansi.Color("\x1b[1;"+color+"m", s) },
	"title": func(title string) string { return ansi.Bold(title) + ansi.BoldDim(": ") },
}

func newTemplate(name, t string) *template.Template {
	return template.Must(template.New(name).Funcs(templFuncs).Parse(t))
}

func execTemplate(t *template.Template, wr io.Writer, v any) {
	if err := t.Execute(wr, v); err != nil {
		panic(err)
	}
}

func getDesc(cmd string) string {
	if Commands == nil || Commands[cmd] == nil {
		return ""
	}
	return Commands[cmd].ShortDescription
}

func sortAliases(aliases []string) {
	slices.SortFunc(aliases, func(a, b string) int {
		la, lb := len(a), len(b)
		switch {
		case la == lb:
			return cmp.Compare(a, b)
		case la < lb:
			return -1
		}
		return 1
	})
}
