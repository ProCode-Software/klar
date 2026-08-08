package ansi

import (
	"bytes"
	"testing"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no tags",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "tag with 1 color",
			input: "<r>hello</r>",
			want:  "\x1b[31mhello\x1b[0m",
		},
		{
			name:  "tag with multiple colors",
			input: "<r bold>hello</>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "nested colors",
			input: "<r>hello <b>world</b>!</r>",
			want:  "\x1b[31mhello \x1b[34mworld\x1b[0m\x1b[31m!\x1b[0m",
		},
		{
			name:  "escaped",
			input: "\\<r>hello",
			want:  "<r>hello",
		},
		{
			name:  "escaped close",
			input: "<r>hello\\</r>",
			want:  "\x1b[31mhello</r>",
		},
		{
			name:  "escaped backslash",
			input: "\\\\<r>hello",
			want:  "\\\x1b[31mhello",
		},
		{
			name:  "close all",
			input: "<r>hello</>",
			want:  "\x1b[31mhello\x1b[0m",
		},
		{
			name:  "close multiple",
			input: "<r bold>hello</r bold>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "close 1 of multiple",
			input: "<r bold>hello</r>",
			want:  "\x1b[31;1mhello\x1b[0m\x1b[1m",
		},
		{
			name:  "close in different order",
			input: "<r bold>hello</bold r>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "actually close all",
			input: "<r bold><dim><c>hello</>",
			want:  "\x1b[31;1m\x1b[2m\x1b[36mhello\x1b[0m",
		},
		{
			name:  "complex nesting",
			input: "<r>A<g>B<b>C</b>D</g>E</r>",
			want:  "\x1b[31mA\x1b[32mB\x1b[34mC\x1b[0m\x1b[31m\x1b[32mD\x1b[0m\x1b[31mE\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Colorize(tt.input)
			if got != tt.want {
				t.Errorf("Colorize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
	t.Run("DisableColor", func(t *testing.T) {
		old := DisableColor
		DisableColor = true
		defer func() { DisableColor = old }()

		input := "<r>hello <b>world</b>!</r>"
		want := "hello world!"
		got := Colorize(input)
		if got != want {
			t.Errorf("Colorize(%q) with DisableColor=true = %q, want %q", input, got, want)
		}
	})
	t.Run("Panic on invalid code", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		Colorize("<unknown>hello")
	})
	t.Run("Panic on invalid closing code", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		Colorize("<r>hello</unknown>")
	})
}

func TestTagFprintf(t *testing.T) {
	buf := &bytes.Buffer{}
	n, err := TagFprintf(buf, "Hello <r bold>%s</>", "world")
	if err != nil {
		t.Fatalf("Fprintf failed: %v", err)
	}
	want := "Hello \x1b[31;1mworld\x1b[0m"
	if buf.String() != want {
		t.Errorf("Fprintf output = %q, want %q", buf.String(), want)
	}
	if n != len(want) {
		t.Errorf("Fprintf n = %d, want %d", n, len(want))
	}
}
