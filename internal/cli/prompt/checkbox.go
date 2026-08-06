package prompt

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"golang.org/x/term"
)

// Bash implementation: https://askubuntu.com/a/1386907
func Checkboxes(title, desc string, opts map[string]bool, def bool) {
	// Prompt
	if title != "" {
		ansi.TagPrintfln("<y>◆</y> <**>%s</**>", title)
	}
	if desc != "" {
		fmt.Println(desc)
	}
	ansi.TagPrintfln(
		"<d><**>Arrows ↑↓</**> Move up/down | <**>Space</**> Toggle | " +
			"<**>A</**> Select/deselect all | <**>Enter</**> Done</d>",
	)

	prevState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), prevState)

	sortedOpts := slices.Sorted(maps.Keys(opts))
	const (
		currChar    = '›'
		checkboxOn  = '▣'
		checkboxOff = '▢'
	)
	var curr int
	for {
		for i, label := range sortedOpts {
			var checkbox string
			if opts[label] {
				checkbox = ansi.BrightGreen(string(checkboxOn))
				label = ansi.Bold(label)
			} else {
				checkbox = ansi.Gray(string(checkboxOff))
			}

			chevron := " "
			if curr == i {
				chevron = ansi.BrightCyan(string(currChar))
				label = ansi.BrightCyan(label)
			}
			fmt.Printf("%s %s %s\r\n", chevron, checkbox, label)
		}
		buf := make([]byte, 3)
		// Read a space, enter, or \e
		if _, err := io.ReadFull(os.Stdin, buf[:1]); err != nil {
			return
		}
		switch buf[0] {
		case ' ': // Select
			opts[sortedOpts[curr]] = !opts[sortedOpts[curr]]
		case 'a', 'A':
			def = !def
			for i := range sortedOpts {
				opts[sortedOpts[i]] = def
			}
		case '\n', '\r':
			return
		case 3, 4, 26:
			return
		case '\x1b': // Arrow keys: read 2 more
			if _, err := io.ReadFull(os.Stdin, buf[1:3]); err != nil {
				return
			}
			switch string(buf[1:]) {
			case "[A", "[D": // Up/left
				if curr == 0 {
					curr = len(sortedOpts) - 1
				} else {
					curr--
				}
			case "[B", "[C": // Down/right
				if curr == len(sortedOpts)-1 {
					curr = 0
				} else {
					curr++
				}
			}
		}
		// Rerender
		fmt.Printf("\x1b[%dA", len(sortedOpts))
	}
}
