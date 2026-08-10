package klon

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

type rang struct {
	Start, End int
}

func TestMarshall(t *testing.T) {
	type testCase struct {
		name     string
		input    any
		flags    klonflags.Flags
		expected string
		indent   int
	}
	cases := []testCase{
		{name: "Int", input: 67, expected: "67"},
		{name: "Bool", input: false, expected: "false"},
		{
			name:     "HexadecimalUint",
			input:    uint32(0xdeada55),
			expected: "0xdeada55",
			flags:    klonflags.HexadecimalUInt,
		},
		{
			name: "InlineObjects",
			input: struct {
				Kind   string
				Range  rang
				Inline bool
			}{"StringLiteral", rang{1, 2}, true},
			expected: `{ kind: StringLiteral, range: { start: 1, end: 2 }, inline: true }`,
			flags:    klonflags.HexadecimalUInt,
			indent:   -1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.indent == 0 {
				// Provide a negative indent instead to test inline printing
				tc.indent = 4
			}
			out, err := Marshall(tc.input, tc.indent, tc.flags)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.expected = strings.Trim(tc.expected, "\n")
			out = bytes.Trim(out, "\n")
			if string(out) != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, string(out))
			}
		})
	}
}

func TestQuote(t *testing.T) {
	type testCase struct {
		name            string
		input, expected string
	}
	cases := []testCase{
		// Cases where no quoting is needed, but there may be escaping
		{name: "EmptyString", input: "", expected: "''"},
		{name: "CanUnquote", input: "hello", expected: "hello"},
		{name: "WithSingleQuotes", input: "it's raining", expected: `it's raining`},
		{name: "Backslash", input: `foo\bar`, expected: `foo\\bar`},
		{name: "Unicode", input: "💀", expected: "💀"},
		{name: "InvalidUTF8", input: "fa\xffhh", expected: `fa\u{ff}hh`},
		{name: "DollarSign", input: "$50", expected: `\$50`},

		// Quoting needed
		{name: "LeadingSpaces", input: "   klar", expected: "'   klar'"},
		{name: "LeadingUnicodeSpace", input: "\u00a0klar", expected: `'\u{a0}klar'`}, // NBSP
		{name: "ANSISequence", input: "\x1b[31m", expected: `\e[31m`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := Quote([]byte(tc.input))
			if string(out) != tc.expected {
				t.Fatalf("expected %#q, got %#q", tc.expected, string(out))
			}
		})
	}
}
