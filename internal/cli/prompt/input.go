package prompt

import (
	"bufio"
	"bytes"
	"fmt"
	"maps"
	"os"
	"slices"
	"unicode"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"golang.org/x/term"
)

func Confirm(msg string, defaultRes bool) bool {
	// (y/n) display
	var defaultStr string
	if defaultRes {
		defaultStr = ansi.ColorSprintf(ansi.CodeDim, "(%s/n)", ansi.BoldBrightGreen("Y"))
	} else {
		defaultStr = ansi.ColorSprintf(ansi.CodeDim, "(y/%s)", ansi.BoldBrightRed("N"))
	}

	fmt.Printf("%s %s: ", msg, defaultStr) // Prompt
	defer fmt.Println()                    // Final newline

	// The terminal has to be made raw so we can read a single character without
	// the user pressing Enter
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}
	for {
		res := make([]byte, 1)
		os.Stdin.Read(res)
		switch bytes.ToLower(res)[0] {
		case ' ', '\n', '\t', 'r':
			continue
		case 'y', 't', '1':
			return true
		case 'n', 'f', '0':
			return false
		default:
			return defaultRes
		}
	}
}

func ChooseLetter(msg string, opts map[byte]string, def byte) byte {
	fmt.Print(msg, ansi.Dim(" ["))
	for i, char := range slices.Sorted(maps.Keys(opts)) {
		if i > 0 {
			fmt.Print(ansi.Dim(" | "))
		}
		label := opts[char]
		var bold string
		if char == def {
			bold = "<** y!>"
			char = byte(unicode.ToUpper(rune(char)))
		}
		ansi.TagPrintf("%s<y>(%c)</y> %s</>", bold, char, label)
	}
	fmt.Print(ansi.Dim("] "))
	for {
		// No raw mode
		buf, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			return def
		}
		got := bytes.ToLower(buf)[0]
		if got == def || got == ' ' || got == '\n' ||
			got == '\t' || got == '\x1b' || got == '\r' {
			return def
		}
		if _, ok := opts[got]; ok {
			return got
		}
		ansi.ColorPrint(ansi.CodeRed, "Please choose a valid option: ")
	}
}
