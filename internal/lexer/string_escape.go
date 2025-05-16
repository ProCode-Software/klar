package lexer

import "unicode"

type StringEscapeType int

const (
	CharacterEscape StringEscapeType = iota
	HexadecimalEscape
	UnicodeEscape
	StringInterpolation
)

type StringEscapeErrorType int

const (
	_ StringEscapeErrorType = iota
	EscapeNotEnough
	EscapeTooLong
	EscapeExpectedChar
	EscapeExpectedHexChar
	EscapeUnknown
)

type stringEscape struct {
	Type          StringEscapeType
	Value         string
	Invalid       bool
	InvalidReason StringEscapeErrorType
}

func (l *Lexer) parseStringEscape(delim rune) stringEscape {
	var (
		escType       StringEscapeType
		isInvalid     bool
		invalidReason StringEscapeErrorType
		isDone        bool
		isHex         = func(r rune) bool { return unicode.Is(unicode.ASCII_Hex_Digit, r) }
	)

	esc := l.TokenizeFunc(func(r rune, s *string) {
		switch {
		case isDone:
			return
		case *s == "":
			switch r {
			case 'u':
				// Unicode escape \u{1234}
				escType = UnicodeEscape

			case 'x':
				// Hexadecimal escape \x12
				escType = HexadecimalEscape
			case '{':
				// Interpolation (double-quoted only)
				if delim == '"' {
					isDone = true
				}
				fallthrough
			case delim, '\\', 'b', 'e', 'f', 'n', 'r', 't':
				// Character escape
				isDone = true
			default:
				// Invalid escape
				isInvalid = true
				invalidReason = EscapeUnknown
				isDone = true
			}
			*s += string(r)
			return
		case *s == "u" && r != '{':
			isInvalid = true
			invalidReason = EscapeExpectedChar
		case escType == HexadecimalEscape:
			if len(*s) >= 2 {
				isDone = true
			} else if !isHex(r) {
				isInvalid = true
				invalidReason = EscapeExpectedHexChar
			}
		case escType == UnicodeEscape && r == '}':
			if len(*s) <= 2 {
				isInvalid = true
				invalidReason = EscapeNotEnough
			}
			isDone = true
		case escType == UnicodeEscape:
			if !isHex(r) {
				isInvalid = true
				invalidReason = EscapeExpectedHexChar
			}
		}
		*s += string(r)
	})
	if escType == UnicodeEscape && len(esc) > 9 { // u + { + 6 + }
		isInvalid = true
		invalidReason = EscapeTooLong
	}
	return stringEscape{
		Type:    escType,
		Value:   esc,
		Invalid: isInvalid,
		InvalidReason: invalidReason,
	}
}