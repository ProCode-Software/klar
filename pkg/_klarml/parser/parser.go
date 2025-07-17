package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
)

type parser struct {
	Index  int
	Tokens []Token
	Errors []error
}

func (p *parser) Current() Token {
	return p.Tokens[p.Index]
}

func (p *parser) CurrentKind() TokenType {
	return p.Tokens[p.Index].Kind
}

func (p *parser) Shift() Token {
	current := p.Current()
	p.Index++
	return current
}

func (p *parser) HasTokens() bool {
	return p.Index < len(p.Tokens)-1
}

func (p *parser) Peek() Token {
	return p.Tokens[p.Index+1]
}

func (p *parser) Error(err error) {
	p.Errors = append(p.Errors, err)
}

func (p *parser) Expect(kind TokenType) Token {
	tok := p.Shift()
	if tok.Kind != kind {
		p.Error(ExpectedTokenErr{kind, tok})
	}
	if !p.HasTokens() {
		p.Index = len(p.Tokens) - 1
	}
	return tok
}

func (p *parser) ExpectDashes(n int) bool {
	got := 0
	index := p.Index
	for p.Tokens[index].Kind == Hyphen {
		got++
		index++
	}
	if got == n {
		p.Index = index
		return true
	}
	return false
}

func (p *parser) RemoveComments() (comments []*ast.Comment) {
	for i := 0; i < len(p.Tokens); i++ {
		curr := p.Tokens[i]
		if curr.Kind != TokenComment {
			continue
		}
		attrs := curr.Attributes.(CommentAttrs)
		if attrs.Unterminated {
			p.Error(UntermCommentErr{curr})
		}
		comments = append(comments, &ast.Comment{
			Block:   attrs.Block,
			Content: curr.Source,
		})
		p.Tokens = slices.Delete(p.Tokens, i, i+1)
		i--
	}
	return comments
}
