package klarml

import (
	"fmt"
	"strconv"
)

func (p *parser) Parse() (d Document, errors []error) {
	for p.HasTokens() {
		var res Value
		switch p.CurrentKind() {
		case Newline:
			continue
		case Dollar:
			decl := p.ParseVarDecl()
			d.Variables = append(d.Variables, decl)
			continue
		case Identifier:
			res = p.ParseObject(0)
		case Numeric:
			res = p.ParseNumber()
		default:
			res = p.ParseValue()
		}
		d.Body = res
	}
	return Document{}, p.Errors
}

func (p *parser) ParseValue() Value {
	var res Value
	switch curr := p.Current(); curr.Kind {
	case String:
		res = p.ParseString()
	case TokenNamespace:
		p.Shift()
		res = Namespace{Name: curr.Source}
	case Dollar:
		res = p.ParseVar()
	case Identifier:
		src := curr.Source
		if src == "true" || src == "false" {
			res = BoolLiteral{Value: src == "true"}
			break
		}
		fallthrough
	default:
		p.Error(UnexpectedTokenErr{curr})
	}
	if p.CurrentKind() == Comma {
		array := Array{Inline: true}
		array.Items = append(array.Items, res)
		for p.CurrentKind() == Comma {
			p.Shift()
			array.Items = append(array.Items, p.ParseValue())
		}
		return array
	}
	return res
}

func (p *parser) ParseNumber() NumericLiteral {
	num := p.Expect(Numeric).Source
	val, err := strconv.ParseFloat(num, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse number %s: %v", num, err))
	}
	return NumericLiteral{Value: val}
}

func (p *parser) ParseString() StringLiteral {
	curr := p.Shift()
	var quoteStyle QuoteStyle
	attrs := curr.Attributes.(StringAttrs)
	switch {
	case attrs.Unquoted:
		quoteStyle = Unquoted
	case attrs.QuoteStyle == '\'':
		quoteStyle = SingleQuote
	case attrs.QuoteStyle == '"':
		quoteStyle = DoubleQuote
	}
	return StringLiteral{
		Content:    curr.Source,
		QuoteStyle: quoteStyle,
	}
}

func (p *parser) ParseVar() (ref VarRef) {
	p.Expect(Dollar)
	if p.CurrentKind() == LeftBrace {
		p.Shift()
		ref.Identifier = p.Expect(Identifier).Source
		ref.Braced = true
		p.Expect(RightBrace)
	} else {
		ref.Identifier = p.Expect(Identifier).Source
	}
	return ref
}

func (p *parser) InvalidMix(isArray bool) {
	pos := p.Current().Position
	p.Error(MixPropAndArrayErr{pos, isArray})
}

func (p *parser) ParseObject(elev int) Value {
	var (
		isArray, isObject bool
		obj               Object
		array             Array
	)
	for p.ExpectDashes(elev) {
		if p.CurrentKind() == Newline {
			continue
		}
		if !isObject && p.Peek().Kind == Colon {
			isObject = true
			if isArray {
				p.InvalidMix(true)
				continue
			}
			prop := Property{Key: p.Expect(Identifier).Source}
			p.Shift()
			if p.CurrentKind() == Newline {
				p.Shift()
				prop.Value = p.ParseObject(elev + 1)
			} else {
				prop.Value = p.ParseValue()
			}
			obj.Properties = append(obj.Properties, prop)
		} else {
			isArray = true
			if isObject {
				p.InvalidMix(false)
			}
		}
		if p.CurrentKind() != EOF {
			array.Items = append(array.Items, p.ParseValue())
			p.Expect(Newline)
		}
	}
	return obj
}

func (p *parser) ParseVarDecl() (decl VarDecl) {
	p.Expect(Dollar)
	decl.Name = p.Expect(Identifier).Source
	p.Expect(Colon)
	if p.CurrentKind() == Newline {
		p.Shift()
		decl.Value = p.ParseObject(1)
	} else {
		decl.Value = p.ParseValue()
	}
	return decl
}
