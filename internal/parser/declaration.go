package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseTypeDeclaration() ast.TypeDeclaration {
	p.Expect(lexer.Type)
	var (
		name = p.Expect(lexer.Identifier)
		/* inherited string
		isEnum bool
		fields []any */
	)

	switch p.Advance().Kind {
	case lexer.Equal:
		// Type
		return ast.TypeAliasDeclaration{
			Identifier: name.Source,
			Type:       p.ParseType(DefaultBindingPower, false),
		}
	case lexer.Colon:
		// Inherited struct
		_ = p.Expect(lexer.Identifier).Source
		fallthrough
	case lexer.LeftCurlyBrace:
		// Struct or enum
		for p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Advance()
		}
		p.Expect(lexer.RightCurlyBrace)
	default:
		// Some other token or unassigned type (if EOS)
		panic(errors.NewTokenError(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
	}
	return nil
}

func (p *Parser) ParseFuncDeclaration() ast.FunctionDeclaration {
	p.Expect(lexer.Func)
	f := ast.FunctionDeclaration{}
	f.Identifier = p.Expect(lexer.Identifier).Source

	// Struct receiver
	// 	func Person.greet()
	if p.CurrentTokenKind() == lexer.Dot {
		p.Advance()
		f.Struct = ast.TypeAlias{Identifier: f.Identifier}
		f.Identifier = p.Expect(lexer.Identifier).Source
	}
	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrentTokenKind() == lexer.LessThan {
		generics := []string{}
		p.Advance()
		for p.IsNot(lexer.GreaterThan) {
			generics = append(generics, p.Expect(lexer.Identifier).Source)
			if p.CurrentTokenKind() != lexer.GreaterThan {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(lexer.GreaterThan)
		f.GenericParams = generics
	}
	// Params
	p.Expect(lexer.LeftParenthesis)
	for p.IsNot(lexer.RightParenthesis) {
		param := ast.FunctionParam{
			Identifier: p.Expect(lexer.Identifier).Source,
		}
		// Optional label:
		// 	func replace(src, with replacement: String)
		if p.CurrentTokenKind() == lexer.Identifier {
			param.Label = param.Identifier
			param.Identifier = p.Expect(lexer.Identifier).Source
		}
		// Parse type: still allow trailing type (example above)
		if p.CurrentTokenKind() == lexer.Colon {
			p.Advance()
			param.Type = p.ParseType(FunctionTypeBindingPower, true).(ast.SimpleType)
		}
		// Default value:
		// 	func List.join(by by: String = ", ")
		if p.CurrentTokenKind() == lexer.Equal {
			p.Advance()
			param.Default = p.ParseExpression(DefaultBindingPower)
		}

		f.Parameters = append(f.Parameters, param)
		if p.CurrentTokenKind() != lexer.RightParenthesis {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)

	// Return type: the arrow. Can be inferred
	if p.CurrentTokenKind() == lexer.Arrow {
		p.Advance()
		f.ReturnType = p.ParseType(DefaultBindingPower, true).(ast.SimpleType)
	}

	// Body: Externally implemented functions may not have a body
	//	@external(js: "./date.js", name: "now")
	// 	func Date.now() -> Date
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		p.Advance()
		for p.IsNot(lexer.RightCurlyBrace) {
			f.Body = append(f.Body, p.ParseStatement())
		}
		p.Expect(lexer.RightCurlyBrace)
	}
	return f
}
