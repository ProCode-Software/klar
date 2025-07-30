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

func (d *Decoder) ReadValue() (lit ast.Value, err error) {
	if err := d.SkipSpace(); err != nil {
		checkEOF(&err)
		return nil, err
	}
	curr := d.Curr()
	switch curr {
	case '\'', '"':
		// String literal
		return d.readString()
	case '+', '-', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.readNumber()
	case '[':
		return d.readArray()
	case ']':
		_, err = d.Advance()
		if err != nil {
			return lit, err
		}
		return nil, ErrUnexpectedBracket
	case '{': // TODO: object
	}
	if unicode.IsSpace(rune(d.Curr())) {
		if err := d.SkipSpaceNewline(); err != nil {
			if err == EOF {
				return &ast.Null{}, nil
			}
			return lit, err
		}
	}
	// TODO: check for - if array or object
	return d.readUnquotedString(false)
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

func (d *Decoder) readArray() (*ast.Array, error) {
	_, err := d.Advance()
	if err != nil {
		return nil, ErrUnterminatedArray
	}
	a := &ast.Array{}

	oldComma := d.CommaSep
	defer func() { d.CommaSep = oldComma }() // Restore
	d.CommaSep = true

	for d.Curr() != ']' {
		val, err := d.ReadValue()
		if err != nil {
			return a, err
		}
		a.Items = append(a.Items, val)
		if d.Curr() != ']' {
			if err := d.Expect(','); err != nil {
				return a, err
			}
		}
	}
	return a, nil
}

func (d *Decoder) readNumber() (ast.Value, error) {
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
		if !isDigit(first) {
			return &ast.String{Value: string(first), Quote: 0}, err
		}
		return value(), err
	}
	for {
		c := d.Curr()
		switch {
		case c == '_' && wasUnderscore, c == '.' && isDecimal:
			isNumber = false
		case c == '_':
			wasUnderscore = true
		case c == '.':
			isDecimal = true
		case unicode.IsSpace(rune(c)), d.isPunct(c):
			return value(), nil
		case !isDigit(c): // letter
			isNumber = false
		}
		b.WriteByte(c)
		if _, err := d.Advance(); err != nil {
			checkEOF(&err)
			return value(), err
		}
		if !isNumber {
			val, err := d.readUnquotedString(true)
			b.WriteString(val.(*ast.String).Value)
			return value(), err
		}
	}
	// return value(), nil
}

func (d *Decoder) isPunct(c byte) bool {
	switch c {
	case '\n', '@', '$', ']', '}':
		return true
	case ',':
		return d.CommaSep
	}
	return false
}

func (d *Decoder) readUnquotedString(continued bool) (ast.Value, error) {
	var s strings.Builder
	value := func() ast.Value {
		str := strings.TrimSpace(s.String())
		if !continued && (str == "true" || str == "false") {
			return &ast.Bool{Value: str == "true"}
		}
		return &ast.String{Value: str, Quote: 0}
	}
	for {
		c := d.Curr()
		if d.isPunct(c) {
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
		r := rune(d.Curr())
		if unicode.IsLetter(r) || unicode.IsDigit(r) ||
			r == '_' || (r == '-' && b.Len() > 0) {
			b.WriteRune(r)
			if _, err := d.Advance(); err != nil {
				return b.String(), err
			}
			continue
		}
		break
	}
	return b.String(), nil
}
