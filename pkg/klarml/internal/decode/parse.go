package decode

import (
	"errors"
	"strings"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

var (
	ErrUnterminatedString = errors.New("unterminated string literal")
	ErrUnterminatedArray  = errors.New("expected '[' to end array")
	ErrUnexpectedBracket  = errors.New("unexpected ']'")
)

func (d *Decoder) ReadValue() (lit ast.Value, err error) {
	if err := d.SkipSpace(); err != nil {
		return nil, err
	}
	switch curr := d.Curr(); curr {
	case '\'', '"':
		// String literal
		return d.readString()
	case 't', 'f':
		lits := map[byte]string{
			't': "rue",
			'f': "alse",
		}
		rest := lits[curr]
		got, err := d.ReadN(len(rest))
		if err != nil {
			return lit, err
		}
		if string(got) == rest {
			return &ast.Bool{Value: curr == 't'}, nil
		}
	case '-', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.readNumber()
	case '[':
		return d.readArray()
	case ']':
		_, err = d.Advance()
		if err != nil {
			return lit, err
		}
		return nil, ErrUnexpectedBracket
	}
	return d.readUnquotedString()
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
	return ret(err)
}

func (d *Decoder) readArray() (*ast.Array, error) {
	_, err := d.Advance()
	if err != nil {
		return nil, ErrUnterminatedArray
	}
	a := &ast.Array{}
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

func (d *Decoder) readNumber() (ast.Value, error) { return nil, nil }

func (d *Decoder) readUnquotedString() (ast.Value, error) {
	var s strings.Builder
	for d.Curr() != '\n' {
		c, err := d.Advance()
		if err != nil {
			// Don't care if there is an EOF
			return &ast.String{Value: s.String()}, nil
		}
		s.WriteByte(c)
	}
	str := s.String()
	if str == "true" || str == "false"  {
		return &ast.Bool{Value: str == "true"}, nil
	}
	return &ast.String{Value: str}, nil
}
