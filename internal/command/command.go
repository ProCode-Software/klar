package command

import (
	"fmt"
	"io"
	"os"
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

func newTemplate(name, t string) *template.Template {
	return template.Must(template.New(name).Funcs(templFuncs).Parse(t))
}

func (c *Command) Help(f io.Writer) {
	t := newTemplate("help", fullHelpTemplate)
	template.Must(t, t.Execute(f, newHelper(c)))
}

func (c *Command) ArgUsage() string {
	if c.Usage == nil {
		c.Usage = c.Flags.Pattern
	}
	var w strings.Builder
	fmt.Fprint(&w, ansi.Bold("Usage")+ansi.BoldDim(": "),
		ansi.BoldGreen(ExecName), " ", ansi.BoldYellow(c.Name),
	)
	for _, arg := range c.Usage {
		fmt.Fprint(&w, ansi.Cyan(" "+arg))
	}
	return w.String()
}

func (c *Command) SeeAlsoString(indent int) string {
	b := &strings.Builder{}
	tw := cli.NewTabWriterOutput(b)
	tw.Margin = indent
	for _, cmd := range c.SeeAlso {
		tw.Write(ansi.BoldGreen(ExecName) + " " + ansi.BoldYellow(cmd), getDesc(cmd))
	}
	tw.Flush()
	return b.String()
}

type helper struct {
	*Command
	ExecName string
}

func newHelper(c *Command) *helper {
	return &helper{
		Command:  c,
		ExecName: ExecName,
	}
}

var templFuncs = template.FuncMap{
	"join": strings.Join,
}

func (helper) ANSI(color, s string) string   { return ansi.Color("\x1b["+color+"m", s) }
func (h helper) Bold(color, s string) string { return ansi.Color("\x1b[1;"+color+"m", s) }
func (h helper) Title(title string) string   { return ansi.Bold(title) + ansi.BoldDim(": ") }
func (helper) FormatExecName() string        { return ansi.BoldGreen(ExecName) }
func (helper) Join(items []string, color, sep string) string {
	c := ansi.Partial("\x1b[" + color + "m")
	return ansi.Color(color, strings.Join(items, ansi.Reset()+sep+c))
}

func getDesc(cmd string) string {
	if Commands == nil || Commands[cmd] == nil {
		return ""
	}
	return Commands[cmd].ShortDescription
}
