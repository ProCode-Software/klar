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
	num, attrs := ReadNumber(l, first)
	return NewToken(pos, Numeric, num).withAttrs(map[string]any{
		"params": attrs,
	})
}

func ReadNumber(rd RuneReader, first rune) (string, NumberAttrs) {
	var (
		b            strings.Builder
		format       IntFormat
		flags        NumberFlags
		errorType    NumberErrorCode
		errPos       int
		isExp, isDec bool
		last         rune
	)
	newError := func(code NumberErrorCode, b *strings.Builder) {
		if errorType == 0 {
			errorType = code
			if b != nil {
				errPos = b.Len()
			}
		}
	}

	b.WriteRune(first)
	last = first

	if first == '0' {
		if r, err := rd.CurrRune(); err == nil {
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
	} else if first == '.' {
		isDec = true
		flags |= IsFloat
	}

	for {
		r, err := rd.CurrRune()
		if err != nil {
			break
		}

		isDigit := IsDigit(r)
		isHexDigit := IsHex(r)

		var stop bool
		switch {
		case r == '0', r == '1':
			// OK for all formats
		case r >= '2' && r <= '9':
			if format == NumberFormatBinary {
				newError(ErrIntIncompatibleDigit, &b)
			}
		case (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F'):
			if r == 'e' || r == 'E' {
				if format == NumberFormatDecimal {
					if isExp {
						stop = true
					} else {
						if last == '_' {
							newError(ErrIntMisplacedSeparator, &b)
							errPos--
						}
						isExp = true
						flags |= HasExponent | IsFloat
					}
					break
				}
			}
			if format != NumberFormatHex {
				stop = true
			}
		case r == '.':
			if isDec || isExp || format != NumberFormatDecimal {
				stop = true
				break
			}
			if last == '_' {
				newError(ErrIntMisplacedSeparator, &b)
				errPos--
			}
			next, err2 := rd.PeekRune()
			if err2 != nil || !IsDigit(next) {
				stop = true
				break
			}
			isDec = true
			flags |= IsFloat
		case r == '_':
			if last == '_' || (format == NumberFormatDecimal && !IsDigit(last)) {
				newError(ErrIntMisplacedSeparator, &b)
			}
			flags |= HasSeparator
		case r == '+', r == '-':
			if (last == 'e' || last == 'E') && format == NumberFormatDecimal {
				// Valid in decimal exponent
			} else {
				stop = true
			}
		default:
			stop = true
		}

		if stop {
			break
		}

		// Re-check digit validity for binary/hex if not already handled
		if isDigit && format == NumberFormatBinary && r > '1' {
			// Already handled
		} else if isHexDigit && format != NumberFormatHex && !(format == NumberFormatDecimal && (r == 'e' || r == 'E')) {
			// Already handled or stop=true
		}

		b.WriteRune(r)
		rd.AdvanceRune()
		last = r
	}

	num := b.String()
	if last == '_' {
		newError(ErrIntMisplacedSeparator, nil)
		errPos = len(num) - 1
	} else if format != NumberFormatHex && !IsDigit(last) {
		newError(ErrIntIncompatibleDigit, nil)
		errPos = len(num)
	}
	var err *NumberError
	if errorType != 0 {
		err = &NumberError{Code: errorType, Offset: uint32(errPos)}
	}
	return num, NumberAttrs{Format: format, Flags: flags, Error: err}
}
