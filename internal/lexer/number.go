package lexer

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

func (l *Lexer) ReadNumber(pos Position) *Token {
	var (
		format       IntFormat
		flags        NumberFlags
		errorType    NumberErrorCode
		errPos       int
		isExp, isDec bool
		last         rune
	)
	newError := func(code NumberErrorCode, b *Builder) {
		errorType = code
		if b != nil {
			errPos = b.Len()
		}
	}
	l.Backup()
	t := l.NewTokenizer(true)
readNumber:
	for r, b := range t.Tokenize {
		// 0 prefix
		if b.String() == "0" {
			switch r {
			case 'x':
				format = NumberFormatHex
				goto writeAndContinue
			case 'b':
				format = NumberFormatBinary
				goto writeAndContinue
			default:
				format = NumberFormatDecimal
			}
		}
		switch r {
		case 'e', 'E':
			// Exponent or hex digit
			if format == NumberFormatDecimal {
				if isExp {
					newError(ErrIntIncompatibleDigit, b)
					break
				}
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				isExp = true
				flags |= HasExponent | IsFloat
				break
			}
			fallthrough // Hex or invalid digit
		case 'a', 'A', 'b', 'B', 'c', 'C', 'd', 'D', 'f', 'F':
			if format != NumberFormatHex {
				// Hex letter or e on other format
				newError(ErrIntIncompatibleDigit, b)
			}
		case '+', '-': // After 'e'
			if last != 'e' && last != 'E' {
				if !IsDigit(last) {
					// 12e+-
					newError(ErrIntIncompatibleDigit, b)
				}
				break readNumber
			}
		case '.':
			switch {
			case isDec:
				break readNumber
			case format != NumberFormatDecimal:
				newError(ErrIntIncompatibleDigit, b)
			case last == '_':
				newError(ErrIntMisplacedSeparator, b)
				errPos--
			}
			if n, isEOF := l.BackupPeek(); isEOF || !IsDigit(rune(n)) {
				break readNumber
			}
			isDec = true
			flags |= IsFloat
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if last == '_' || (format == NumberFormatDecimal && !IsDigit(last)) {
				newError(ErrIntMisplacedSeparator, b)
			}
			flags |= HasSeparator
		default:
			switch {
			case !IsDigit(r):
				break readNumber
			case format == NumberFormatDecimal,
				format == NumberFormatHex,
				format == NumberFormatBinary && r <= '1':
			default:
				newError(ErrIntIncompatibleDigit, b)
			}
		}
	writeAndContinue:
		b.WriteRune(r)
		last = r
	}
	num := t.String()
	// Last character validation
	if last == '_' {
		// Last digit can't be a separator
		newError(ErrIntMisplacedSeparator, nil)
		errPos = len(num) - 1
	} else if format != NumberFormatHex && !IsDigit(last) {
		// "1e-" .. EOF
		newError(ErrIntIncompatibleDigit, nil)
		errPos = len(num)
	}
	var err *NumberError
	if errorType != 0 {
		err = &NumberError{Code: errorType, Offset: uint32(errPos)}
	}
	return NewToken(pos, Numeric, num).withAttrs(attrs{
		"params": NumberAttrs{Format: format, Flags: flags, Error: err},
	})
}
