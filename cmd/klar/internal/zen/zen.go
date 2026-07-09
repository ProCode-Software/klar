package zen

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/util"
	"golang.org/x/term"
)

// TODO: Write
const Zen = ``

var width = 80

func Run(c *command.Runner) {
	// Wrap the text around the terminal's width
	var err error
	if width, _, err = term.GetSize(int(os.Stdout.Fd())); err != nil {
		width = 80
	}

	// Title
	fmt.Print(ansi.Bold(center("The Zen of Klar")), "\n")

	// 1. Wrap
	var b strings.Builder
	util.Wrap(Zen, &b, width, width, 0)
	wrapped := b.String()

	// 2. Center
	b.Reset()
	for line := range strings.SplitSeq(wrapped, "\n") {
		b.WriteString(center(line))
		b.WriteByte('\n')
	}

	// 3. Color + print
	fmt.Print(util.KlarGradient(b.String()))
}

func center(s string) string {
	ln := utf8.RuneCountInString(s)
	if ln >= width {
		return s
	}
	half := (width - ln) / 2
	return string(char.Repeat(' ', half)) + s
}

const LongDescription = `Klar is built around specific idioms and principles. It is important for all developers to follow them. When you first learned Klar, you may have seen The Zen of Klar before, but it's always great to get a refresher once in a while. So whenever you need a reminder about Klar's principles and idioms, run 'klar zen'!`
