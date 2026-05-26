package lexer

import (
	"strings"
)

type NumberAttrs struct {
	Format IntFormat
	Flags  NumberFlags
	Error  *NumberError
}

type IntFormat uint8

const (
	NumberFormatDecimal IntFormat = iota
	NumberFormatHex               // 0x
	NumberFormatBinary            // 0b
)

type NumberFlags uint8

const (
	IsFloat      NumberFlags = 1 << iota // .
	HasSeparator                         // _
	HasExponent                          // e
)

type NumberError struct {
	Code   NumberErrorCode
	Offset uint32
}

type NumberErrorCode uint8

const (
	_ NumberErrorCode = iota

	ErrIntMisplacedSeparator // '_' must separate digits
	ErrIntIncompatibleDigit  // Invalid digit for the current format
	ErrInvalidDecimalPoint   // Decimal point can only be used in decimal (base 10) format
)

func (l *Lexer) ReadNumber(pos Position, first rune) *Token {
	num, params := ReadNumber(l, first)
	return NewToken(pos, Numeric, num).withAttrs(attrs{"params": params})
}

func ReadNumber(rd RuneReader, first rune) (string, NumberAttrs) {
	var (
		b            strings.Builder
		format       IntFormat
		flags        NumberFlags
		err          *NumberError
		isExp, isDec bool
		last         rune
	)
	newError := func(code NumberErrorCode, errPos int) {
		if err == nil {
			err = &NumberError{Code: code, Offset: uint32(errPos)}
		}
	}

	b.WriteRune(first)
	last = first

	if first == '0' {
		if r, er := rd.CurrRune(); er == nil {
			switch r {
			case 'x', 'X':
				format = NumberFormatHex
				b.WriteRune(r)
				rd.AdvanceRune()
				last = r
			case 'b', 'B':
				format = NumberFormatBinary
				b.WriteRune(r)
				rd.AdvanceRune()
				last = r
			default:
				format = NumberFormatDecimal
			}
		}
	}

readNumber:
	for {
		r, er := rd.CurrRune()
		if er != nil {
			break
		}
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if format == NumberFormatBinary && r > '1' {
				newError(ErrIntIncompatibleDigit, b.Len())
			}
		case 'e', 'E':
			// Exponent or hex digit
			if format == NumberFormatDecimal {
				if isExp {
					newError(ErrIntIncompatibleDigit, b.Len())
					break
				}
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b.Len()-1)
				}
				isExp = true
				flags |= HasExponent | IsFloat
				break
			}
			fallthrough // Hex or invalid digit
		case 'a', 'b', 'c', 'd', 'f', 'A', 'B', 'C', 'D', 'F':
			if format != NumberFormatHex {
				// Hex letter or e on other format
				newError(ErrIntIncompatibleDigit, b.Len())
			}
		case '.':
			if isDec || isExp || format != NumberFormatDecimal {
				break readNumber
			}
			if last == '_' {
				newError(ErrIntMisplacedSeparator, b.Len()-1)
			}
			// Check if next character is a digit
			next, err2 := rd.PeekRune()
			if err2 != nil || !IsDigit(next) {
				break readNumber
			}
			isDec = true
			flags |= IsFloat
		case '_':
			if last == '_' || (format == NumberFormatDecimal && !IsDigit(last)) {
				newError(ErrIntMisplacedSeparator, b.Len())
			}
			flags |= HasSeparator
		case '+', '-':
			if (last != 'e' && last != 'E') || format != NumberFormatDecimal {
				break readNumber
			}
		default:
			break readNumber
		}

		b.WriteRune(r)
		rd.AdvanceRune()
		last = r
	}

	num := b.String()
	if last == '_' {
		newError(ErrIntMisplacedSeparator, len(num)-1)
	} else if format != NumberFormatHex && !IsDigit(last) {
		newError(ErrIntIncompatibleDigit, len(num))
	}
	return num, NumberAttrs{Format: format, Flags: flags, Error: err}
}
