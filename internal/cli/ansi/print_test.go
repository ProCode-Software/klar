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
			name:  "simple color",
			input: "<red>hello</red>",
			want:  "\x1b[31mhello\x1b[0m",
		},
		{
			name:  "multiple colors space separated",
			input: "<red bold>hello</>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "nested colors",
			input: "<red>hello <blue>world</blue>!</red>",
			want:  "\x1b[31mhello \x1b[34mworld\x1b[0m\x1b[31m!\x1b[0m",
		},
		{
			name:  "escaping open",
			input: "\\<red>hello",
			want:  "<red>hello",
		},
		{
			name:  "escaping close",
			input: "<red>hello\\</red>",
			want:  "\x1b[31mhello</red>\x1b[0m",
		},
		{
			name:  "escaped backslash",
			input: "\\\\<red>hello",
			want:  "\\\x1b[31mhello\x1b[0m",
		},
		{
			name:  "invalid tag treated as literal",
			input: "<unknown>hello",
			want:  "<unknown>hello",
		},
		{
			name:  "generic close",
			input: "<red>hello</>",
			want:  "\x1b[31mhello\x1b[0m",
		},
		{
			name:  "named close",
			input: "<red>hello</red>",
			want:  "\x1b[31mhello\x1b[0m",
		},
		{
			name:  "mismatched close ignored",
			input: "<red>hello</blue>",
			want:  "\x1b[31mhello\x1b[0m", // Should pop red? Or ignore?
			// Plan said: "Closing: </...> (can be named </red> or generic </>)."
			// "Closing a tag restores the previous state."
			// If we have strict matching, </blue> should be ignored if top is red.
			// Let's assume strict matching for named tags.
		},
		{
			name:  "strict closing match",
			input: "<red bold>hello</red bold>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "loose closing match subset",
			input: "<red bold>hello</red>",
			want:  "\x1b[31;1mhello\x1b[0m", // Should close because red is in {red, bold}
		},
		{
			name:  "loose closing match order",
			input: "<red bold>hello</bold red>",
			want:  "\x1b[31;1mhello\x1b[0m", // Should close because sets match
		},
		{
			name:  "loose closing mismatch",
			input: "<red bold>hello</blue>",
			want:  "\x1b[31;1mhello\x1b[0m", // Should NOT close because blue is not in {red, bold}
		},
		{
			name:  "generic close works for multi",
			input: "<red bold>hello</>",
			want:  "\x1b[31;1mhello\x1b[0m",
		},
		{
			name:  "complex nesting",
			input: "<red>A<green>B<blue>C</blue>D</green>E</red>",
			want:  "\x1b[31mA\x1b[32mB\x1b[34mC\x1b[0m\x1b[31m\x1b[32mD\x1b[0m\x1b[31mE\x1b[0m",
			// Note: When popping blue, we restore red+green.
			// When popping green, we restore red.
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
}

func TestFprintf(t *testing.T) {
	var buf bytes.Buffer
	n, err := Fprintf(&buf, "Hello <red bold>%s</>", "world")
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
