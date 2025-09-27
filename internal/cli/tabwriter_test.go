package cli_test

import (
	"bytes"
	"testing"

	"github.com/ProCode-Software/klar/internal/cli"
)

const expected0 = `a1   b2   cc3    d4
one  two  three
1    2    3      4`

const expected1 = `  a1  b2  cc3  d4
 one twothree
   1   2    3   4`

func setup(tw *cli.TabWriter) (cli.TabWriter, *bytes.Buffer) {
	if tw == nil {
		tw = new(cli.TabWriter)
		tw.Spacing = 2
	}
	buf := &bytes.Buffer{}
	tw.Output = buf
	return *tw, buf
}

func TestNewTabWriter(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		tw, buf := setup(nil)
		tw.Write("a1", "b2", "cc3", "d4")
		tw.WriteBytes([]byte("one\ttwo\tthree\n"))
		tw.WriteString("1\t2\t3\t4\n")
		if _, err := tw.Flush(); err != nil {
			panic(err)
		}
		if str := buf.String(); str != expected0 {
			t.Errorf("want:\n\n%s\n\ngot:\n\n%s\n", expected0, str)
		}
	})
	// TODO: test ansi
	t.Run("Advanced", func(t *testing.T) {
		tw, buf := setup(&cli.TabWriter{Spacing: 0, Flags: cli.AlignRight, MinWidth: 4, Separator: ','})
		tw.Write("a1", "b2", "cc3", "d4")
		tw.WriteBytes([]byte("one,two,three\n"))
		tw.WriteString("1,2,3,4\n")
		if _, err := tw.Flush(); err != nil {
			panic(err)
		}
		if str := buf.String(); str != expected1 {
			t.Errorf("want:\n\n%s\n\ngot:\n\n%s\n", expected1, str)
		}
	})
}
