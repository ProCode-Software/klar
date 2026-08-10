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
			name: "Object",
			input: struct {
				Kind   string
				Range  rang
				Inline bool
			}{"StringLiteral", rang{1, 2}, true},
			expected: `kind: StringLiteral
range:
    - start: 1
    - end: 2

inline: true`,
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

var quoteCases = []struct {
	name            string
	input, expected string
}{
	// Cases where no quoting is needed, but there may be escaping
	{name: "EmptyString", input: "", expected: "''"},
	{name: "CanUnquote", input: "hello", expected: "hello"},
	{name: "WithSingleQuotes", input: "it's raining", expected: `it's raining`},
	{name: "Backslash", input: `foo\bar`, expected: `foo\\bar`},
	{name: "Unicode", input: "💀", expected: "💀"},
	{
		name:     "DecomposedUnicode",
		input:    "cafe" + string([]byte{0xcc, 0x81}), // Combining Acute Accent
		expected: "cafe\u0301",
	},
	{name: "InvalidUTF8", input: "fa\xffhh", expected: `fa\u{ff}hh`},
	{name: "DollarSign", input: "$50", expected: `\$50`},
	{name: "Tab", input: "\t", expected: `\t`},
	{name: "HasNewline", input: "Hello\nWorld", expected: `Hello\nWorld`},
	{name: "BackslashN", input: `Hello\nWorld`, expected: `Hello\\nWorld`},

	// Quoting needed
	{name: "LeadingSpaces", input: "   klar", expected: "'   klar'"},
	{name: "LeadingUnicodeSpace", input: "\u00a0klar", expected: `'\u{a0}klar'`}, // NBSP
	{name: "ANSISequence", input: "\x1b[31m", expected: `'\e[31m'`},

	// Strings that contain invalid UTF-8
	// Source: https://stackoverflow.com/a/3886015
	{
		name:     "InvalidUTF8",
		input:    "\xfc\xa1\xa1\xa1\xa1\xa1",
		expected: `\u{fc}\u{a1}\u{a1}\u{a1}\u{a1}\u{a1}`,
	},
	{
		name:     "InvalidUTF8-2",
		input:    "\xf0\x90\x28\xbc",
		expected: `\u{f0}\u{90}\u{28}\u{bc}`,
	},
	{name: "InvalidUTF8-3", input: "\xc3\x28", expected: `\u{c3}(`},
}

func TestQuote(t *testing.T) {
	for _, tc := range quoteCases {
		t.Run(tc.name, func(t *testing.T) {
			out := Quote([]byte(tc.input))
			if string(out) != tc.expected {
				t.Fatalf("expected %#q, got %#q", tc.expected, string(out))
			}
		})
	}
}

func TestQuoteDeterminism(t *testing.T) {
	for _, tc := range quoteCases {
		t.Run(tc.name, func(t *testing.T) {
			marsh, err := Marshall(tc.input, 0)
			if err != nil {
				t.Fatalf("failed to marshall input: %v", err)
			}
			var unmarsh string
			if err := Unmarshall(marsh, &unmarsh); err != nil {
				t.Fatalf("failed to unmarshall: %v", err)
			}
			if unmarsh != tc.input {
				t.Fatalf(
					"expected Unmarshall to return input %#v, got %#v",
					tc.input, unmarsh,
				)
			}
		})
	}
}
