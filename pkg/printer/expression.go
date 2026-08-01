package printer

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *printer) printExpression(expr ast.Expression) {
	switch expr := expr.(type) {
	case *ast.Symbol:
		p.writeString(expr.Identifier, expr.Range)
	case *ast.FloatLiteral:
		p.writeString(expr.Source, expr.Range)
	case *ast.IntegerLiteral:
		p.writeString(expr.Source, expr.Range)
	case *ast.BooleanLiteral:
		if expr.Value == true {
			p.writeString("true", expr.Range)
		} else {
			p.writeString("false", expr.Range)
		}
	case *ast.NilLiteral:
		p.writeString(lexer.Nil.String(), expr.Range)
	case *ast.BinaryExpression:
		p.printExpression(expr.Left)
		p.writeString(expr.Operator.String(), expr.Operator.Range())
		p.printExpression(expr.Right)
	case *ast.RelationalExpression:
		p.printExpression(expr.Expressions[0])
		for i, operand := range expr.Expressions[1:] {
			op := expr.Operators[i]
			p.writeString(op.String(), op.Range())
			p.printExpression(operand)
		}
	case *ast.AssertExpression:
		p.printExpression(expr.Expression)
		op := lexer.NotNot.String()
		p.writeStringAt(op, expr.Range.End.Sub(0, uint32(len(op))))
	case *ast.AwaitExpression:
		p.writeStringAt(lexer.Await.String(), expr.Range.Start)
		p.printExpression(expr.Expression)
	case *ast.GoExpression:
		p.writeStringAt(lexer.Go.String(), expr.Range.Start)
		if expr.Body != nil {
			p.printNode(expr.Body)
		} else {
			p.printExpression(expr.Expression)
		}
	case *ast.Discard:
		p.writeString(lexer.Underscore.String(), expr.Range)
	default:
		panic(fmt.Sprintf("unhandled expression: %T", expr))
	}
}
