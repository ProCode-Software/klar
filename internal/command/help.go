package command

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/pkg/argparse"
	"golang.org/x/term"
)

var termWidth int

func formatCmd(subCommand string) string {
	return ansi.BoldMagenta(ExecName) + " " + ansi.BoldYellow(subCommand)
}

func (c *Command) Help(f io.Writer) {
	if f, ok := f.(*os.File); ok {
		var err error
		termWidth, _, err = term.GetSize(int(f.Fd()))
		if err != nil {
			termWidth = 80
		}
	}
	execTemplate(newTemplate("help", fullHelpTemplate), f, c)
}

func (c *Command) ArgUsage() string {
	var w strings.Builder
	execTemplate(newTemplate("usage", usageTemplate), &w, c)
	return w.String()
}

func (c *Command) AliasesString() string {
	var b strings.Builder
	b.Grow(10 + len(c.Aliases)*4)
	b.WriteString(ansi.Bold("Aliases"))
	b.WriteString(ansi.BoldDim(": "))
	for i, alias := range c.Aliases {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(formatCmd(alias))
	}
	return b.String()
}

func (c *Command) SeeAlsoString(indent int) string {
	b := &strings.Builder{}
	tw := cli.NewTabWriterOutput(b)
	tw.ReserveCapacity(len(c.SeeAlso), 2)
	tw.Margin = indent
	for _, cmd := range c.SeeAlso {
		tw.WriteCells(formatCmd(cmd), getDesc(cmd))
	}
	tryFlush(tw)
	return b.String()
}

func (c *Command) FlagString(indent int) string {
	if c.Flags == nil {
		return ""
	}
	var (
		b       = &strings.Builder{}
		tw      = cli.NewTabWriterOutput(b)
		defs    = c.Flags.FlagDefs
		aliases = make(map[string][]string, len(defs))
		cyan    = ansi.Partial(ansi.CodeCyan)
		sep     = ansi.Reset() + ", " + cyan
	)
	tw.Margin = indent
	tw.Spacing, tw.WrapIndent = 4, 4
	tw.ReserveCapacity(len(defs), 2)
	tw.TermWidth = termWidth
	tw.Flags |= cli.WrapTerminalColumns
	// Make the alias map
	for alias, actual := range c.Flags.FlagAliases {
		aliases[actual] = append(aliases[actual], alias)
	}
	// Print each line
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		flag := c.Flags.FlagDefs[name]
		al := aliases[name]
		sortAliases(al) // Sort aliases by length
		if len(al) > 0 && len(al[0]) == 1 {
			// Short alias
			tw.WriteString(
				cyan + argparse.FormatFlag(al[0]) + sep + argparse.FormatFlag(name),
			)
			al = al[1:]
		} else {
			tw.WriteString("    " + cyan + argparse.FormatFlag(name))
		}
		for _, alias := range al {
			tw.WriteString(sep + argparse.FormatFlag(alias))
		}
		if flag.ParamName != "" {
			tw.WriteString(ansi.Green(" <" + flag.ParamName + ">"))
		}
		fmt.Fprintf(tw, "%s\t%s %s\n", ansi.Reset(), flag.Description, getDefault(flag))
	}
	tryFlush(tw)
	return b.String()
}

func tryFlush(tw *cli.TabWriter) {
	if _, err := tw.Flush(); err != nil {
		cli.InternalError("Failed to flush output while showing help: ", err)
	}
}

func getDefault(flag argparse.FlagDef) string {
	var def any
	switch {
	case flag.Default == nil:
		return ""
	case flag.Type == argparse.TypeEnum:
		def = flag.Default.Enum().Key()
		if def == "" {
			return ""
		}
	default:
		switch v := flag.Default.Value; v {
		case "", false, 0, nil:
			return ""
		default:
			def = v
		}
	}
	return ansi.Gray(fmt.Sprintf("(default: %v)", def))
}

var templFuncs = template.FuncMap{
	"join":      strings.Join,
	"hasPrefix": strings.HasPrefix,
	"wrap": func(s string) string {
		b := strings.Builder{}
		b.Grow(len(s) + len(s)/termWidth)
		cli.Wrap(s, &b, termWidth, 0, 0)
		return b.String()
	},

	"exec":  func() string { return ansi.BoldMagenta(ExecName) },
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
