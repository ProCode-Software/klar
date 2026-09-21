package jswriter

import (
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (w *Writer) writeExpression(expr jsir.Expression) {
	switch expr := expr.(type) {
	case *jsir.ArrayLiteral:
	case *jsir.ArrowFunction:
	case *jsir.AssignmentExpression:
		w.writeAssignmentExpr(expr, false)
	case *jsir.BinaryExpression:
		prec := expr.Operator.Precedence()
		w.writeExprWithPrec(expr.Left, prec)
		w.writeByte(' ')
		w.writeString(expr.Operator.String())
		w.writeByte(' ')
		w.writeExprWithPrec(expr.Right, prec)
	case *jsir.BooleanLiteral:
		if expr.Value == true {
			w.writeString("true")
		} else {
			w.writeString("false")
		}
	case *jsir.CallExpression:
		w.writeCallExpr(expr)
	case *jsir.ClassDeclaration:
		w.writeClass(expr)
	case *jsir.CommaExpression:
	case *jsir.FunctionDeclaration:
		w.writeFunction(expr, true)
	case *jsir.MemberExpression:
	case *jsir.NullLiteral:
		w.writeString("null")
	case *jsir.NumericLiteral:
		// TODO: Ensure JavaScript's number format is compatible with Go's
		// Using 'g' instead of 'f', large numbers will be written
		// in scientific notation
		w.writeString(strconv.FormatFloat(expr.Value, 'g', -1, 64))
	case *jsir.ObjectLiteral:
	case *jsir.PostfixUnaryExpression:
		w.writeString(expr.Operator.String())
		// Should always be false since the operator is alway '++' or '--'
		if expr.Operator.ShouldAddSpace() {
			w.writeByte(' ')
		}
		w.writeExprWithPrec(expr.Operand, expr.Operator.Precedence())
	case *jsir.RegExpLiteral:
		w.writeByte('/')
		w.writeString(expr.Pattern)
		w.writeByte('/')
		if expr.Flags != "" {
			w.writeString(expr.Flags)
		}
	case *jsir.SpreadExpression:
		w.writeString("...")
		w.writeExprWithPrec(expr.Right, 0) // TODO: Precedence
	case *jsir.StringLiteral:
		w.writeStringLiteral(expr)
	case *jsir.Symbol:
		w.writeString(expr.Name)
	case *jsir.TernaryExpression:
		// TODO: Precedence
		w.writeExprWithPrec(expr.Condition, 0)
		w.writeString(" ? ")
		w.writeExprWithPrec(expr.True, 0)
		w.writeString(" : ")
		w.writeExprWithPrec(expr.False, 0)
	case *jsir.TemplateLiteral:
	case *jsir.UnaryExpression:
		w.writeString(expr.Operator.String())
		if expr.Operator.ShouldAddSpace() {
			w.writeByte(' ')
		}
		w.writeExprWithPrec(expr.Operand, expr.Operator.Precedence())
	case *jsir.UndefinedLiteral:
		w.writeString("undefined")
	default:
		panic(fmt.Sprintf("unhandled expression: %T", expr))
	}
}

func (w *Writer) writeAssignmentExpr(expr *jsir.AssignmentExpression, firstOnLine bool) {
	// let res = f()
	// ;[item] = res
	if firstOnLine {
		if _, ok := expr.Assignee.(*jsir.ArrayLiteral); ok {
			w.writeByte(';')
		}
	}
	// TODO: Parentheses will be needed if the destructure is an object, or
	// the assignment is being used in an expression. If parens are added,
	// and this is an expression statement, add a semicolon.
	w.writeExpression(expr.Assignee)
	w.writeByte(' ')
	w.writeString(expr.Operator.String())
	w.writeByte(' ')
	w.writeExpression(expr.Value)
}

// writeExprWithPrec wraps the expression in parentheses as needed.
func (w *Writer) writeExprWithPrec(expr jsir.Expression, parentPrec int) {
	var thisPrec int
	if expr, ok := expr.(jsir.Precedencer); ok {
		thisPrec = expr.Precedence()
	}
	needParen := thisPrec != 0 && thisPrec < parentPrec
	if needParen {
		w.writeByte('(')
	}
	w.writeExpression(expr)
	if needParen {
		w.writeByte(')')
	}
}

func (w *Writer) writeStringLiteral(str *jsir.StringLiteral) {
	// TODO: Characters should be escaped and perform auto quoting
	quote := str.QuoteStyle
	if quote == jsir.AutoQuote {
		quote = jsir.SingleQuote
	}
	w.writeByte(byte(quote))
	w.writeString(str.Content)
	w.writeByte(byte(quote))
}

func (w *Writer) writeFunction(fn *jsir.FunctionDeclaration, funcKeyword bool) {
	// TODO: Parentheses are needed if the function is at the top-level
	// or is being called
	if (fn.Modifiers & jsir.AsyncFunction) != 0 {
		w.writeString("async ")
	}
	if funcKeyword {
		w.writeString("function")
		if (fn.Modifiers & jsir.GeneratorFunction) != 0 {
			w.writeByte('*')
		}
	}
	if fn.Name != "" {
		if funcKeyword {
			w.writeByte(' ')
		}
		w.writeString(fn.Name)
	}

	w.writeByte('(')
	for i, param := range fn.Params {
		if i > 0 {
			w.writeString(", ")
		}
		w.writeFuncParam(param)
	}
	if fn.VariadicParam != nil {
		if len(fn.Params) > 0 {
			w.writeString(", ")
		}
		w.writeString("...")
		w.writeFuncParam(fn.VariadicParam)
	}
	w.writeByte(')')

	if fn.ReturnType != nil {
		w.writeString(": ")
		w.writeTSType(fn.ReturnType)
	}
	// TypeScript function signatures don't have bodies
	if fn.Body != nil {
		w.writeByte(' ')
		w.writeBlock(fn.Body)
	}
}

func (w *Writer) writeFuncParam(param *jsir.FunctionParam) {
	w.writeExpression(param.Name)
	if param.Type != nil {
		w.writeString(": ")
		w.writeTSType(param.Type)
	}
	if param.Default != nil {
		w.writeString(" = ")
		w.writeExpression(param.Default)
	}
}

func (w *Writer) writeClass(cls *jsir.ClassDeclaration) {
	w.writeString("class ")
	if cls.Name != "" {
		w.writeString(cls.Name)
		w.writeByte(' ')
	}
	if cls.Extends != "" {
		w.writeString("extends ")
		w.writeString(cls.Extends)
		w.writeByte(' ')
	}
	w.writeString("{\n")
	w.increaseLevel()
	for _, field := range cls.Fields {
		w.writeIndent()
		mods := cls.Flags[field]
		if (mods & jsir.Static) != 0 {
			w.writeString("static ")
		}
		w.writeBinding(field)
		w.writeByte('\n')
	}
	for _, meth := range cls.Methods {
		w.writeIndent()
		mods := cls.Flags[meth]
		if (mods & jsir.Static) != 0 {
			w.writeString("static ")
		}
		if (mods & jsir.Getter) != 0 {
			w.writeString("get ")
		} else if (mods & jsir.Setter) != 0 {
			w.writeString("set ")
		}
		w.writeFunction(meth, false)
		w.writeByte('\n')
	}
	if cls.StaticInit != nil {
		w.writeIndent()
		w.writeString("static ")
		w.writeBlock(cls.StaticInit)
		w.writeByte('\n')
	}
	w.decreaseLevel()
	w.writeByte('}')
}

func (w *Writer) writeCallExpr(call *jsir.CallExpression) {
	w.writeExpression(call.Callee)
	w.writeByte('(')
	for i, arg := range call.Arguments {
		if i > 0 {
			w.writeString(", ")
		}
		w.writeExprWithPrec(arg, 0)
	}
	w.writeByte(')')
}
