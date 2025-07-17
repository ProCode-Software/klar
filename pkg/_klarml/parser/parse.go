package parser

import (
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

func ParseTokens(tokens []Token) (d *ast.Document, errors []error) {
	
	parserTokens := make([]Token, len(tokens))
	copy(parserTokens, tokens)
	p := parser{
		Index:  0,
		Tokens: parserTokens,
	}
	return p.Parse()
}

func (p *parser) Parse() (doc *ast.Document, errors []error) {
	doc = &ast.Document{Comments: p.RemoveComments()}
	for p.HasTokens() {
		var res ast.Value
		switch p.CurrentKind() {
		case Newline:
			continue
		case Dollar:
			decl := p.ParseVarDecl()
			doc.Variables = append(doc.Variables, decl)
			continue
		case Identifier:
			res = p.ParseObject(0)
		case Numeric:
			res = p.ParseNumber()
		default:
			res = p.ParseValue()
		}
		doc.Body = res
	}
	return doc, p.Errors
}

func (p *parser) ParseValue() ast.Value {
	var res ast.Value
	switch curr := p.Current(); curr.Kind {
	case String:
		res = p.ParseString()
	case TokenNamespace:
		p.Shift()
		res = &ast.Namespace{Name: curr.Source}
	case Dollar:
		res = p.ParseVar()
	case Identifier:
		src := curr.Source
		if src == "true" || src == "false" {
			res = &ast.BoolLiteral{Value: src == "true"}
			break
		}
		fallthrough
	default:
		p.Error(UnexpectedTokenErr{curr})
	}
	if p.CurrentKind() == Comma {
		array := &ast.Array{Inline: true}
		array.Items = append(array.Items, res)
		for p.CurrentKind() == Comma {
			p.Shift()
			array.Items = append(array.Items, p.ParseValue())
		}
		return array
	}
	return res
}

func (p *parser) ParseNumber() *ast.NumericLiteral {
	num := p.Expect(Numeric).Source
	val, err := strconv.ParseFloat(num, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse number %s: %v", num, err))
	}
	return &ast.NumericLiteral{Value: val}
}

func (p *parser) ParseString() *ast.StringLiteral {
	curr := p.Shift()
	var quoteStyle ast.QuoteStyle
	attrs := curr.Attributes.(StringAttrs)
	switch {
	case attrs.Unquoted:
		quoteStyle = ast.Unquoted
	case attrs.QuoteStyle == '\'':
		quoteStyle = ast.SingleQuote
	case attrs.QuoteStyle == '"':
		quoteStyle = ast.DoubleQuote
	}
	return &ast.StringLiteral{
		Content:    curr.Source,
		QuoteStyle: quoteStyle,
	}
}

func (p *parser) ParseVar() *ast.VarRef {
	p.Expect(Dollar)
	ref := &ast.VarRef{}
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

func (p *parser) ParseObject(elev int) ast.Value {
	var (
		isArray, isObj bool
		obj            = &ast.Object{}
		array          = &ast.Array{}
	)
	for p.ExpectDashes(elev) {
		if p.CurrentKind() == Newline {
			continue
		}
		if !isObj && p.Peek().Kind == Colon {
			isObj = true
			if isArray {
				p.InvalidMix(true)
				continue
			}
			prop := &ast.Property{Key: p.Expect(Identifier).Source}
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
			if isObj {
				p.InvalidMix(false)
			}
		}
		if p.CurrentKind() != EOF {
			array.Items = append(array.Items, p.ParseValue())
			p.Expect(Newline)
		}
	}
	if isArray {
		return array
	}
	return obj
}

func (p *parser) ParseVarDecl() *ast.VarDecl {
	p.Expect(Dollar)
	decl := &ast.VarDecl{Name: p.Expect(Identifier).Source}
	p.Expect(Colon)
	if p.CurrentKind() == Newline {
		p.Shift()
		decl.Value = p.ParseObject(1)
	} else {
		decl.Value = p.ParseValue()
	}
	return decl
}
