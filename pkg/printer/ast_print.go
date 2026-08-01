package printer

import (
	"bytes"
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// PrintNode formats the provided node as it was written in the source.
// To pretty-print the node, use KlarFormat instead.
func PrintNode(node ast.Node) []byte {
	p := &printer{pos: node.GetRange().Start}
	p.printNode(node)
	return p.buf.Bytes()
}

type printer struct {
	buf      bytes.Buffer
	pos      lexer.Position
	comments []*ast.Comment
}

func (p *printer) writeNewline() {
	p.printComments(lexer.Position{p.pos.Line + 1, 1})
}

func (p *printer) writeString(s string, rang ranges.Range) {
	p.printComments(rang.Start)
	p.buf.WriteString(s)
	p.pos = rang.End
}

func (p *printer) writeStringAt(s string, start lexer.Position) {
	p.printComments(start)
	p.buf.WriteString(s)
	p.pos = start.Add(0, uint32(len(s)))
}

// TODO: Print comments from the program (not required to be passed)
func (p *printer) printNode(node ast.Node) {
	defer p.printComments(node.GetRange().End)
	switch node := node.(type) {
	case *ast.Program:
		p.comments = node.Comments
		for i, stmt := range node.Body {
			if i > 0 {
				p.writeNewline()
			}
			p.printStatement(stmt)
		}
	case ast.Statement:
		p.printStatement(node)
	case ast.Expression:
		p.printExpression(node)
	case ast.Type:
		p.printType(node)
	case *ast.Block:
		p.writeStringAt(lexer.LeftCurlyBrace.String(), node.Range.Start)
		for i, stmt := range node.Body {
			if i > 0 {
				p.writeNewline()
			}
			p.printStatement(stmt)
		}
		rb := lexer.RightCurlyBrace.String()
		p.writeStringAt(rb, node.Range.End.Sub(0, uint32(len(rb))))
	case ast.Identifier:
		p.writeString(node.Name, node.Range())
	case *ast.CallParam:
	default:
		panic(fmt.Sprintf("unhandled node: %T", node))
	}
}

// Writes comments and spaces until p.pos == to
func (p *printer) printComments(to lexer.Position) {
	if p.pos == to {
		return
	}
	// Comments
	// TODO: < or <=?
	for len(p.comments) > 0 && ranges.ComparePos(p.comments[0].Range.Start, to) < 0 {
		cmt := p.comments[0]
		start := cmt.Range.Start
		p.writeSpace(start)
		if cmt.Type == lexer.BlockComment {
			p.writeStringAt("/*", start)
			p.writeString(cmt.Value, ranges.Range{start.Add(0, 2), cmt.Range.End.Sub(0, 2)})
			p.writeStringAt("*/", cmt.Range.End.Sub(0, 2))
		} else {
			p.writeStringAt("//", start)
			p.writeString(cmt.Value, ranges.Range{start.Add(0, 2), cmt.Range.End})
		}
	}
	// Spaces
	p.writeSpace(to)
}

func (p *printer) writeSpace(to lexer.Position) {
	for p.pos.Line < to.Line {
		p.buf.WriteByte('\n')
		p.pos.Line++
		p.pos.Col = 1
	}
	p.buf.Write(char.Repeat('\n', int(to.Col)-int(p.pos.Col)))
}
