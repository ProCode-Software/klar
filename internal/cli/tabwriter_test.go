package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProCode-Software/klar/internal/cli"
)

const expectedDefault = `
a1   b2   cc3    d4
one  two  three
1    2    3      4`

const expectedRight = `
  a1  b2  cc3  d4
 one twothree
   1   2    3   4`

func setup(tw *cli.TabWriter) (cli.TabWriter, *bytes.Buffer) {
	if tw == nil {
		tw = &cli.TabWriter{Spacing: 2}
	}
	buf := &bytes.Buffer{}
	tw.Output = buf
	return *tw, buf
}

func TestTabWriter(t *testing.T) {
	t.Run("DefaultSettings", func(t *testing.T) {
		tw, buf := setup(nil)
		tw.WriteCells("a1", "b2", "cc3", "d4")
		tw.Write([]byte("one\ttwo\tthree\n"))
		tw.WriteString("1\t2\t3\t4\n")
		if _, err := tw.Flush(); err != nil {
			panic(err)
		}
		exp := strings.Trim(expectedDefault, "\n")
		if str := buf.String(); strings.Trim(str, "\n") != exp {
			t.Errorf("want:\n%s\ngot:\n%s", border(t, exp), border(t, str))
		}
	})
	// TODO: test ansi
	t.Run("CommaSep_AlignRight", func(t *testing.T) {
		tw, buf := setup(&cli.TabWriter{
			Spacing: 0, Flags: cli.AlignRight, MinWidth: 4, Separator: ',',
		})
		tw.WriteCells("a1", "b2", "cc3", "d4")
		tw.Write([]byte("one,two,three\n"))
		tw.WriteString("1,2,3,4\n")
		if _, err := tw.Flush(); err != nil {
			panic(err)
		}
		exp := strings.Trim(expectedRight, "\n")
		if str := buf.String(); strings.Trim(str, "\n") != exp {
			t.Errorf("want:\n%s\ngot:\n%s", border(t, exp), border(t, str))
		}
	})
}

func border(t *testing.T, s string) string {
	t.Helper()
	var b strings.Builder
	var i, lastLineLen int
	for line := range strings.Lines(s) {
		if i == 0 {
			b.Write(bytes.Repeat([]byte{'-'}, len(line)+1))
			b.WriteByte('\n')
		}
		b.WriteByte('|')
		line := strings.TrimSuffix(line, "\n")
		b.WriteString(line)
		b.WriteByte('|')
		b.WriteByte('\n')
		i++
		lastLineLen = len(line)
	}
	b.Write(bytes.Repeat([]byte{'-'}, lastLineLen+2))
	return b.String()
}
