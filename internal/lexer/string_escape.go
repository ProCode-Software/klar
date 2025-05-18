package lexer

import (
	"unicode"
)

type StringEscapeType int

const (
	_ StringEscapeType = iota
	CharacterEscape
	HexadecimalEscape
	UnicodeEscape
	StringInterpolation
)

type StringEscapeErrorType int

const (
	_ StringEscapeErrorType = iota
	ErrEscapeNotEnough
	ErrEscapeTooLong
	ErrEscapeExpectedChar
	ErrEscapeExpectedHexChar
	ErrEscapeUnknown
)

type StringEscape struct {
	Type          StringEscapeType
	Value         string
	Invalid       bool
	InvalidReason StringEscapeErrorType
}

// Intentionally creating a new lexer
func (l Lexer) parseStringEscape(pos Position, delim rune) StringEscape {
	var (
		escType       StringEscapeType
		isInvalid     bool
		invalidReason StringEscapeErrorType
		isDone        bool
	)
	pos = Position{pos.Line, pos.Col - 1}
	isHex := func(r rune) bool { return unicode.Is(unicode.ASCII_Hex_Digit, r) }

	esc := l.TokenizeFwdFunc(func(r rune, s *string) {
		switch {
		case isDone:
			return
		case *s == "" && r == '{':
			escType = StringInterpolation
		case escType == StringInterpolation && r == '}':
			if len(*s) == 1 {
				isInvalid = true
				invalidReason = ErrEscapeNotEnough
			}
			isDone = true
		case *s == "\\":
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
				invalidReason = ErrEscapeUnknown
				isDone = true
			}
			*s += string(r)
			return
		case *s == "\\u" && r != '{':
			isInvalid = true
			invalidReason = ErrEscapeExpectedChar
		case escType == HexadecimalEscape:
			if len(*s) >= 3 {
				isDone = true
			} else if !isHex(r) {
				isInvalid = true
				invalidReason = ErrEscapeExpectedHexChar
			}
		case escType == UnicodeEscape && r == '}':
			if len(*s) < 4 { // At least 1 digit required
				isInvalid = true
				invalidReason = ErrEscapeNotEnough
			}
			isDone = true
		case escType == UnicodeEscape:
			if !isHex(r) {
				isInvalid = true
				invalidReason = ErrEscapeExpectedHexChar
			}
		}
		*s += string(r)
	})
	if escType == UnicodeEscape && len(esc) > 10 { // \ + u + { + 6 + }
		isInvalid = true
		invalidReason = ErrEscapeTooLong
	}
	return StringEscape{
		Type:          escType,
		Value:         esc,
		Invalid:       isInvalid,
		InvalidReason: invalidReason,
	}
}
