package jswriter

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

// A newline isn't appended.
func (w *Writer) writeStatement(stmt jsir.Statement) {
	switch stmt := stmt.(type) {
	// Basic
	case *jsir.ExpressionStatement:
		// Ensure a semicolon is put before an array destructure
		// TODO: Any type of an expression can start with parentheses.
		// Ensure a semicolon is added before it.
		if assign, ok := stmt.Expression.(*jsir.AssignmentExpression); ok {
			w.writeAssignmentExpr(assign, true)
			break
		}
		w.writeExpression(stmt.Expression)
	case *jsir.BindingDeclaration:
		w.writeString(stmt.Kind.String())
		w.writeByte(' ')
		w.writeBinding(&stmt.Binding)
	case *jsir.FunctionDeclaration:
		w.writeFunction(stmt, true)
	case *jsir.ClassDeclaration:
		w.writeClass(stmt)
	case *jsir.MultiBindingDeclaration:
		w.writeString(stmt.Kind.String())
		w.writeByte(' ')
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
		w.writeForStmt(stmt)
	case *jsir.WhileStatement:
		w.writeString("while (")
		w.writeExpression(stmt.Condition)
		w.writeString(") ")
		w.writeControlBlock(stmt.Body)
	case *jsir.DoWhileStatement:
		w.writeString("do ")
		w.writeControlBlock(stmt.Do)
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
	case *jsir.TryStatement:
		w.writeTryStmt(stmt)
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
		w.writeImportStmt(stmt)
	case *jsir.ExportModifierStatement:
		w.writeString("export ")
		if stmt.Default {
			w.writeString("default ")
		}
		switch obj := stmt.Declaration.(type) {
		case jsir.Statement:
			w.writeStatement(obj)
		case jsir.Expression:
			w.writeExpression(obj)
		default:
			panic(fmt.Sprintf("invalid ExportModifierStatement declaration: %T", obj))
		}
	case *jsir.NamedExportsStatement:
		w.writeString("export ")
		w.writeImportNames(stmt.Exports)
	case *jsir.ExportFromStatement:
		w.writeExportFromStmt(stmt)
	// TODO: TypeScript .dts statements
	default:
		panic(fmt.Sprintf("unhandled statement: %T", stmt))
	}
}

func (w *Writer) writeBlock(body *jsir.Block) {
	if len(body.Statements) == 0 {
		w.writeString("{}")
		return
	}
	// TODO: Should we put a single statement on the same line as the braces?
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
// an if/while/for-statement.
func (w *Writer) writeControlBlock(body *jsir.Block) {
	switch len(body.Statements) {
	case 0:
		w.writeByte(';')
	case 1:
		w.writeStatement(body.Statements[0])
	default:
		w.writeBlock(body)
	}
}

func (w *Writer) writeBinding(b *jsir.Binding) {
	w.writeExpression(b.Variable)
	if b.Type != nil {
		w.writeString(": ")
		w.writeTSType(b.Type)
	}
	if b.Value != nil {
		w.writeString(" = ")
		w.writeExpression(b.Value)
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
	// Always add braces if the body is an if/for/while-statement. Dangling 'else'.
	// 'for' and 'while' statements are also checked because we may have this:
	//
	// 	if (user.username == 'admin')
	// 		for (const [key, value] of user.options)
	// 			if (key == 'firstLoginDate') ...
	// 			else ...
	writeBlock := func(b *jsir.Block) {
		if len(b.Statements) == 1 {
			switch b.Statements[0].(type) {
			case *jsir.IfStatement, *jsir.WhileStatement, *jsir.ForStatement:
				w.writeBlock(b)
				return
			}
		}
		w.writeControlBlock(b)
	}

	writeBlock(s.Then)

	if s.ElseIf != nil && len(*s.ElseIf) > 0 {
		for _, elif := range *s.ElseIf {
			w.writeString(" else if (")
			w.writeExpression(elif.Condition)
			w.writeString(") ")
			writeBlock(elif.Then)
		}
	}
	if s.Else != nil {
		w.writeString(" else ")
		writeBlock(s.Else)
	}
}

func (w *Writer) writeForStmt(stmt *jsir.ForStatement) {
	w.writeString("for (")
	switch stmt.Kind {
	case jsir.ForOfLoop, jsir.ForInLoop:
		w.writeString(stmt.BindingKind.String())
		w.writeByte(' ')
		w.writeExpression(stmt.Variable)
		if stmt.Kind == jsir.ForOfLoop {
			w.writeString(" of ")
		} else {
			w.writeString(" in ")
		}
		w.writeExpression(stmt.Iterator)
	case jsir.ForCLoop:
		if stmt.CForLoop.Init != nil {
			w.writeStatement(stmt.CForLoop.Init)
		}
		w.writeString("; ")
		if stmt.CForLoop.Test != nil {
			w.writeExpression(stmt.CForLoop.Test)
		}
		w.writeString("; ")
		if stmt.CForLoop.Update != nil {
			w.writeExpression(stmt.CForLoop.Update)
		}
	default:
		panic(fmt.Sprintf("unknown ForLoopKind: %d", stmt.Kind))
	}
	w.writeString(") ")
	w.writeControlBlock(stmt.Body)
}

func (w *Writer) writeTryStmt(stmt *jsir.TryStatement) {
	// Braces are always required around the blocks
	w.writeString("try ")
	w.writeBlock(stmt.Try)
	if stmt.Catch != nil {
		w.writeString(" catch ")
		if stmt.CatchExpr != nil {
			w.writeByte('(')
			w.writeExpression(stmt.CatchExpr)
			w.writeString(") ")
		}
		w.writeBlock(stmt.Catch)
	}
	if stmt.Finally != nil {
		w.writeString(" finally ")
		w.writeBlock(stmt.Finally)
	}
	if stmt.Catch == nil && stmt.Finally == nil {
		panic("both Catch and Finally are nil in jsir.TryStatement")
	}
}

func (w *Writer) writeImportName(imp jsir.ImportName) {
	w.writeString(imp.Name)
	if imp.As != "" {
		w.writeString(" as ")
		w.writeString(imp.As)
	}
}

func (w *Writer) writeImportNames(names []jsir.ImportName) {
	if len(names) == 0 {
		w.writeString("{}")
		return
	}
	w.writeString("{ ")
	for i, imp := range names {
		if i > 0 {
			w.writeString(", ")
		}
		w.writeImportName(imp)
	}
	w.writeString(" }")
}

func (w *Writer) writeImportStmt(stmt *jsir.ImportStatement) {
	w.writeString("import ")
	if stmt.DefaultImport != nil {
		w.writeString(*stmt.DefaultImport)
		if stmt.NamedImports != nil || stmt.NamespaceImport != nil {
			w.writeString(", ")
		}
	}
	switch {
	case stmt.NamedImports != nil:
		w.writeImportNames(*stmt.NamedImports)
	case stmt.NamespaceImport != nil:
		w.writeString("* as ")
		w.writeString(*stmt.NamespaceImport)
	case stmt.DefaultImport == nil:
		panic(
			"no type of import (default, namespace, or named) set for jsir.ImportStatement",
		)
	}

	// 'from' isn't needed for 'import "..."'
	if needsFrom := stmt.NamedImports != nil || stmt.NamespaceImport != nil ||
		stmt.DefaultImport != nil; needsFrom {
		w.writeString(" from ")
	}
	w.writeStringLiteral(&jsir.StringLiteral{jsir.AutoQuote, stmt.From})
	if len(stmt.With) > 0 {
		w.writeImportWith(stmt.With)
	}
}

func (w *Writer) writeExportFromStmt(stmt *jsir.ExportFromStatement) {
	w.writeString("export ")
	switch {
	case stmt.Star != nil && *stmt.Star == "":
		w.writeByte('*')
	case stmt.Star != nil:
		w.writeString("* as ")
		w.writeString(*stmt.Star)
	case stmt.NamedExports != nil:
		w.writeImportNames(*stmt.NamedExports)
	default:
		panic(
			"jsir.ExportFromStatement.Star and jsir.ExportFromStatement.NamedExports are both nil",
		)
	}
	w.writeString(" from ")
	w.writeStringLiteral(&jsir.StringLiteral{jsir.AutoQuote, stmt.From})
	if len(stmt.With) > 0 {
		w.writeImportWith(stmt.With)
	}
}

func (w *Writer) writeImportWith(with map[string]jsir.StringLiteral) {
	w.writeString(" with { ")
	var once bool
	for k, v := range with {
		if once {
			w.writeString(", ")
		}
		w.writeString(k) // TODO: Quote
		w.writeString(": ")
		w.writeStringLiteral(&v)
		once = true
	}
	w.writeString(" }")
}
