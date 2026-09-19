package jswriter

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

// A newline isn't appended.
func (w *Writer) writeStatement(stmt jsir.Statement) {
	switch stmt := stmt.(type) {
	// Generic
	case *jsir.ExpressionStatement:
		w.writeExpression(stmt.Expression)
	case *jsir.BindingDeclaration:
		w.writeString(stmt.Kind.String())
		w.writeByte(' ')
		w.writeBinding(&stmt.Binding)
	case *jsir.MultiBindingDeclaration:
		w.writeString(stmt.Kind.String())
		for i, binding := range stmt.Bindings {
			if i > 0 {
				w.writeString(", ")
			}
			w.writeBinding(binding)
		}
	case *jsir.EmptyStatement:
		w.writeByte(';')
	case *jsir.LabelledStatement:
		w.writeString(stmt.Name)
		w.writeString(": ")
		// Prettier puts the label on the same line as the statement
		w.writeStatement(stmt.Statement)
	case *jsir.Block:
		w.writeBlock(stmt)

	// Control Flow
	case *jsir.IfStatement:
		w.writeIfStmt(stmt)
	case *jsir.ThrowStatement:
		w.writeString("throw ")
		w.writeExpression(stmt.Expression)
	case *jsir.SwitchStatement:
		w.writeSwitchStmt(stmt)
	case *jsir.ReturnStatement:
		w.writeString("return")
		if stmt.Expression != nil {
			w.writeByte(' ')
			w.writeExpression(stmt.Expression)
		}
	case *jsir.BreakStatement:
		w.writeString("break")
		if stmt.Label != "" {
			w.writeByte(' ')
			w.writeString(stmt.Label)
		}
	case *jsir.ForStatement:
	case *jsir.WhileStatement:
		w.writeString("while (")
		w.writeExpression(stmt.Condition)
		w.writeString(") ")
		w.writeControlBlock(stmt.Body, false)
	case *jsir.DoWhileStatement:
		w.writeString("do ")
		w.writeControlBlock(stmt.Do, false)
		// In JS, this braces aren't required if the body is block or semicolon
		// Invalid: do f() while (x > 0) -- Add a newline before 'while'
		if len(stmt.Do.Statements) == 1 {
			w.writeByte('\n')
			w.writeIndent()
		} else {
			w.writeByte(' ')
		}
		w.writeString("while (")
		w.writeExpression(stmt.While)
		w.writeString(")")
	case *jsir.ContinueStatement:
		w.writeString("continue")
		if stmt.Label != "" {
			w.writeByte(' ')
			w.writeString(stmt.Label)
		}
	case *jsir.DebuggerStatement:
		w.writeString("debugger")

	// Import/Export
	case *jsir.ImportStatement:
	case *jsir.ExportModifierStatement:
		w.writeString("export ")
		if stmt.Default {
			w.writeString("default ")
		}
		w.writeExpression(stmt.Declaration)
	case *jsir.NamedExportsStatement:
	case *jsir.ExportFromStatement:

	// TODO: TypeScript .dts statements
	default:
		panic(fmt.Sprintf("unhandled statement: %T", stmt))
	}
}

func (w *Writer) writeBlock(body *jsir.Block) {
	if len(body.Statements) == 0 {
		w.writeString("{}")
	}

	w.writeString("{\n")
	w.increaseLevel()
	w.writeStatementList(body.Statements)
	w.decreaseLevel()
	w.writeIndent() // Previous level
	w.writeByte('}')
}

func (w *Writer) writeStatementList(stmts []jsir.Statement) {
	for _, stmt := range stmts {
		w.writeIndent()
		w.writeStatement(stmt)
		w.writeByte('\n')
	}
}

// writeControlBlock adds braces conditionally to the block for use in
// an if/while/for-statement. If ifSafe is true, braces will be added
// if the body is a single if-statement.
func (w *Writer) writeControlBlock(body *jsir.Block, ifSafe bool) {
	switch len(body.Statements) {
	case 0:
		w.writeByte(';')
	case 1:
		if ifSafe && len(body.Statements) == 1 {
			if _, ok := body.Statements[0].(*jsir.IfStatement); ok {
				w.writeBlock(body)
				return
			}
		}
		w.writeStatement(body.Statements[0])
	default:
		w.writeBlock(body)
	}
}

func (w *Writer) writeBinding(b *jsir.Binding) {
	w.writeDestructure(b.Variable)
	if b.Type != nil {
		w.writeString(": ")
		w.writeTSType(b.Type)
	}
	if b.Value != nil {
		w.writeString(" = ")
		w.writeExpression(b.Value)
	}
}

func (w *Writer) writeDestructure(dest jsir.Destructure) {
	switch dest := dest.(type) {
	case jsir.Expression:
		w.writeExpression(dest)
	default:
		panic(fmt.Sprintf("unhandled destructure: %T", dest))
	}
}

func (w *Writer) writeSwitchStmt(s *jsir.SwitchStatement) {
	// We won't indent 'case', similar to Go formatting.
	const indentCases = false

	w.writeString("switch (")
	w.writeExpression(s.Subject)
	w.writeString(") ")
	if len(s.Cases) == 0 && s.Default == nil {
		w.writeString("{}")
		return
	}

	w.writeString("{\n")
	if indentCases {
		w.increaseLevel()
	}
	for _, cs := range s.Cases {
		w.writeIndent()
		w.writeString("case ")
		w.writeExpression(cs.Expression)
		w.writeString(":\n")
		w.increaseLevel()
		w.writeStatementList(cs.Body.Statements)
		w.writeByte('\n')
	}
	if s.Default != nil {
		w.writeIndent()
		w.writeString("default:\n")
		w.increaseLevel()
		w.writeStatementList(s.Default.Statements)
		w.writeByte('\n')
		w.decreaseLevel()
	}

	if indentCases {
		w.decreaseLevel()
	}
	w.writeIndent() // Previous level
	w.writeByte('}')
}

func (w *Writer) writeIfStmt(s *jsir.IfStatement) {
	w.writeString("if (")
	w.writeExpression(s.Condition)
	w.writeString(") ")
	// Always add braces if the body is an if-statement. Dangling 'else'
	w.writeControlBlock(s.Then, true)

	if s.ElseIf != nil && len(*s.ElseIf) > 0 {
		for _, elif := range *s.ElseIf {
			w.writeString(" else if (")
			w.writeExpression(elif.Condition)
			w.writeString(") ")
			w.writeControlBlock(elif.Then, true)
		}
	}
	if s.Else != nil {
		w.writeString(" else ")
		w.writeControlBlock(s.Else, true)
	}
}
