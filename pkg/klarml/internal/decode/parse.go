package decode

import (
	goerrors "errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

var (
	ErrUnterminatedString = goerrors.New("klarml: unterminated string literal")
	ErrUnterminatedArray  = goerrors.New("klarml: expected ']' to end array")
	ErrUnexpectedBracket  = goerrors.New("klarml: unexpected ']'")
	ErrUnexpectedBrace    = goerrors.New("klarml: unexpected '}'")
)

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// Sets *err to nil if err == io.EOF
func checkEOF(err *error) {
	if *err == EOF {
		*err = nil
	}
}

func (d *Decoder) ReadValue(pf parseFlags) (lit ast.Value, err error) {
	if err := d.SkipSpace(); err != nil {
		checkEOF(&err)
		return nil, err
	}
	curr, start := d.Curr(), d.Offset
	switch curr {
	case '\'', '"':
		// String literal
		lit, err = d.readString()
	case '+', '-', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		lit, err = d.readNumber(pf)
	case '[':
		lit, err = d.readArray(pf)
	case ']':
		_, err = d.Advance()
		if err != nil {
			return lit, err
		}
		lit, err = &ast.String{Value: "]"}, ErrUnexpectedBracket
	case '{': // TODO: object

	default:
		if d.Curr() != '\n' {
			lit, err = d.readUnquotedString(pf)
			break
		}
		// Value on newline or object/array
		switch err = d.SkipSpaceNewline(); err {
		case nil:
		case EOF:
			lit = &ast.Nil{} // Nil if EOF
		default:
			return lit, err
		}
		next, err := d.ReadN(2)
		switch err {
		case nil:
			// Object or array (could also be a number if followed by digit)
			if next[0] == '-' && !isDigit(next[1]) {
				// TODO: Read object/array
				break
			}
			fallthrough
		case EOF: // Value on newline
			lit, err = d.ReadValue(pf)
		default:
			return lit, err
		}
	}
	if lit != nil {
		lit.SetRange(start, d.Offset)
	}
	return lit, err
}

func (d *Decoder) readString() (*ast.String, error) {
	quote, err := d.Advance()
	if err != nil {
		return nil, ErrUnterminatedString
	}
	var b strings.Builder
	ret := func(err error) (*ast.String, error) {
		return &ast.String{
			Quote: quote,
			Value: b.String(),
		}, err
	}
	for d.Curr() != quote {
		c, err := d.Advance()
		b.WriteByte(c)
		if err != nil {
			return ret(ErrUnterminatedString)
		}
		if c == '\\' {
			next, err := d.Advance()
			if err != nil {
				return ret(ErrUnterminatedString)
			}
			b.WriteByte(next)
		}
	}
	_, err = d.Advance() // Last token is known quote
	if err == nil || err == EOF {
		return ret(nil)
	}
	return ret(err)
}

func (d *Decoder) readArray(pf parseFlags) (*ast.Array, error) {
	_, err := d.Advance() // [
	if err != nil {
		if err == EOF {
			return nil, ErrUnterminatedArray
		}
		return nil, err
	}
	a := &ast.Array{}
	for d.Curr() != ']' {
		val, err := d.ReadValue(pf | comma)
		if err != nil {
			return a, err
		}
		a.Items = append(a.Items, val)
		if d.Curr() != ']' {
			if err := d.ExpectSpacesThen(','); err != nil {
				return a, err
			}
		}
	}
	_, err = d.Advance()
	checkEOF(&err)
	return a, err
}

func (d *Decoder) readNumber(flags parseFlags) (ast.Value, error) {
	var b strings.Builder
	isNumber := true
	var isDecimal, wasUnderscore bool
	value := func() ast.Value {
		src := b.String()
		if !isNumber {
			return &ast.String{Value: src, Quote: 0}
		}
		num, err := strconv.ParseFloat(src, 64)
		if err != nil {
			panic(fmt.Sprintf("can't parse number: %q", src))
			// return &ast.String{Value: src, Quote: 0}
		}
		return &ast.Number{Source: src, Value: num}
	}
	// Check first digit or +, -, .
	first, err := d.Advance()
	b.WriteByte(first)
	if err != nil {
		checkEOF(&err)
		if !isDigit(first) { // + or -
			return &ast.String{Value: string(first), Quote: 0}, err
		}
		return value(), err
	}
	for {
		c, size := d.CurrRune()
		switch {
		case c == '_' && wasUnderscore, c == '.' && isDecimal:
			isNumber = false
		case c == '_':
			wasUnderscore = true
		case c == '.':
			isDecimal = true
		case unicode.IsSpace(c), d.isPunct(byte(c), flags):
			return value(), nil
		case !isDigit(byte(c)): // letter
			isNumber = false
		}
		b.WriteRune(c)
		if err := d.AdvanceN(size); err != nil {
			checkEOF(&err)
			return value(), err
		}
		if !isNumber {
			val, err := d.readUnquotedString(flags | continuedString)
			b.WriteString(val.(*ast.String).Value)
			return value(), err
		}
	}
	// return value(), nil
}

func (d *Decoder) isPunct(c byte, f parseFlags) bool {
	switch c {
	case '\n', '@', '$', ']', '}':
		return true
	case ',':
		return f&comma != 0
	}
	return false
}

func (d *Decoder) readUnquotedString(flags parseFlags) (ast.Value, error) {
	var s strings.Builder
	value := func() ast.Value {
		str := strings.TrimSpace(s.String())
		if flags&continuedString == 0 && (str == "true" || str == "false") {
			return &ast.Bool{Value: str == "true"}
		}
		return &ast.String{Value: str, Quote: 0}
	}
	for {
		c := d.Curr()
		if d.isPunct(c, flags) {
			break
		}
		s.WriteByte(c)
		_, err := d.Advance()
		if err != nil {
			checkEOF(&err)
			return value(), err
		}
	}
	return value(), nil
}

// Returns a nil error if another byte can be read
func (d *Decoder) ReadIdent() (string, error) {
	var b strings.Builder
	for {
		r, size := d.CurrRune()
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_', r == '-' && b.Len() > 0:
			b.WriteRune(r)
			if err := d.AdvanceN(size); err != nil {
				return b.String(), err
			}
			continue
		}
		break
	}
	return b.String(), nil
}
