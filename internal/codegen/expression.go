package codegen

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertExpression(expr ast.Expression) jsir.Expression {
	switch expr := expr.(type) {
	case *ast.AssertExpression:
	case *ast.AwaitExpression:
	case *ast.BinaryExpression:
	case *ast.BooleanLiteral:
	case *ast.CallExpression:
	case *ast.EnumLiteral:
	case *ast.ForExpression:
	case *ast.FloatLiteral:
	case *ast.IntegerLiteral:
	case *ast.GoExpression:
	case *ast.IndexExpression:
	case *ast.LambdaExpression:
	case *ast.ListCastExpression:
	case *ast.ListLiteral:
	case *ast.MapCastExpression:
	case *ast.MapLiteral:
	case *ast.NilLiteral:
	case *ast.ObjectPipeline:
	case *ast.PipelineExpression:
	case *ast.ParenExpression:
	case *ast.RangeExpression:
	case *ast.RegexLiteral:
	case *ast.RelationalExpression:
	case *ast.RestExpression:
	case *ast.SliceExpression:
	case *ast.StringLiteral:
	case *ast.StructDotInit:
	case *ast.Symbol:
	case *ast.TryExpression:
	case *ast.TupleLiteral:
	case *ast.UnaryExpression:
	case *ast.WhenExpression:
	default:
		panic(fmt.Sprintf("unhandled ast.Expression type: %T", expr))
	}
	return nil
}
