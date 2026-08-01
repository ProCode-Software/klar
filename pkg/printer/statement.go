package printer

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *printer) printStatement(stmt ast.Statement) {
	comma := lexer.Comma.String()
	switch stmt := stmt.(type) {
	case *ast.ExpressionStatement:
		p.printExpression(stmt.Expression)
	case *ast.AssignmentStatement:
		writeWithSeparator(p, stmt.Assignee, comma)
		p.writeString(stmt.Operator.String(), stmt.Operator.Range())
		writeWithSeparator(p, stmt.Values, comma)
	case *ast.VariableDeclaration:
		writeWithSeparator(p, stmt.Variables, comma)
		if stmt.ExplicitType != nil {
			p.writeStringAt(lexer.Colon.String(), p.pos)
			p.printType(stmt.ExplicitType)
		}
		p.writeStringAt(lexer.ColonEqual.String(), p.pos)
		writeWithSeparator(p, stmt.Values, comma)
	case *ast.NextStatement:
		p.writeStringAt(lexer.Next.String(), stmt.Range.Start)
		if stmt.Label != nil {
			p.writeStringAt(lexer.Colon.String(), stmt.Label.Position.Sub(0, 1))
			p.printNode(stmt.Label)
		}
	case *ast.StopStatement:
		p.writeStringAt(lexer.Stop.String(), stmt.Range.Start)
		if stmt.Label != nil {
			p.writeStringAt(lexer.Colon.String(), stmt.Label.Position.Sub(0, 1))
			p.printNode(stmt.Label)
		}
	case *ast.Attribute:
		p.writeStringAt(lexer.At.String(), stmt.Range.Start)
		p.printNode(stmt.Name)
		if stmt.Args != nil {
			p.writeStringAt(lexer.LeftParenthesis.String(), stmt.Name.End())
			writeWithSeparator(p, stmt.Args, comma)
			p.writeStringAt(lexer.RightParenthesis.String(), stmt.Range.End.Sub(0, 1))
		}
	default:
		panic(fmt.Sprintf("unhandled statement: %T", stmt))
	}
}

func writeWithSeparator[T ast.Node](p *printer, items []T, sep string) {
	for i, node := range items {
		if i > 0 {
			p.writeStringAt(sep, p.pos)
		}
		p.printNode(node)
	}
}
