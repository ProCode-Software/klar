package jswriter

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

// Type Annotations
// =========

func (w *Writer) writeTSType(typ jsir.TSType) {
	switch typ := typ.(type) {
	case *jsir.TSArray:
		w.writeTSType(typ.ItemType)
		w.writeString("[]")
	case *jsir.TSComputedIndex:
		w.writeTSType(typ.Type)
		w.writeByte('[')
		w.writeTSType(typ.Index)
		w.writeByte(']')
	case *jsir.TSFunction:
		w.writeTSFunction(typ, false)
	case *jsir.TSGeneric:
		w.writeTSType(typ.Type)
		for i, arg := range typ.Arguments {
			if i > 0 {
				w.writeString(", ")
			}
			w.writeTSType(arg)
		}
	case *jsir.TSImport:
		w.writeString("import(")
		w.writeStringLiteral(&typ.Path)
		w.writeByte(')')
	case *jsir.TSIntersection:
		// TODO: May need to wrap items in parens based on precedence
		// Also for unions
		for i, item := range typ.Items {
			if i > 0 {
				w.writeString(" & ")
			}
			w.writeTSType(item)
		}
	case *jsir.TSNamespaceIndex:
		for i, ns := range typ.Chain {
			if i > 0 {
				w.writeByte('.')
			}
			w.writeString(ns)
		}
	case *jsir.TSObjectLiteral:
		w.writeTSObjectLiteral(typ)
	case *jsir.TSTuple:
		w.writeByte('[')
		for i, item := range typ.Items {
			if i > 0 {
				w.writeString(", ")
			}
			w.writeTSType(item)
		}
		w.writeByte(']')
	case *jsir.TSUnion:
		for i, item := range typ.Items {
			if i > 0 {
				w.writeString(" | ")
			}
			w.writeTSType(item)
		}
	case jsir.Expression: // Literal type
		w.writeExpression(typ)
	default:
		panic(fmt.Sprintf("unhandled TSType: %T", typ))
	}
}

func (w *Writer) writeTSFunction(typ *jsir.TSFunction, objectMethod bool) {
	w.writeByte('(')
	for i, param := range typ.Arguments {
		if i > 0 {
			w.writeString(", ")
		}
		w.writeTSTypePair(param)
	}
	if objectMethod {
		w.writeString("): ")
	} else {
		w.writeString(") => ")
	}
	w.writeTSType(typ.Return)
}

func (w *Writer) writeTSObjectLiteral(lit *jsir.TSObjectLiteral) {
	// Write 2 or less properties on a single line
	var singleLine bool
	if lit.ComputedProperties != nil {
		singleLine = len(*lit.ComputedProperties)+len(lit.Properties) <= 2
	} else {
		singleLine = len(lit.Properties) <= 2
	}

	if singleLine {
		w.writeString("{ ")
	} else {
		w.writeString("{\n")
		w.increaseLevel()
	}

	for i, prop := range lit.Properties {
		if !singleLine {
			w.writeIndent()
		} else if i > 0 {
			w.writeString(", ")
		}
		w.writeTSTypePair(prop)
	}
	if lit.ComputedProperties != nil {
		for i, prop := range *lit.ComputedProperties {
			if !singleLine {
				w.writeIndent()
			} else if i > 0 && len(lit.Properties) == 0 {
				w.writeString(", ")
			}
			w.writeString("[_: ")
			w.writeTSType(prop[0])
			w.writeString("]: ")
			w.writeTSType(prop[1])
		}
	}

	if singleLine {
		w.writeString(" }")
	} else {
		w.decreaseLevel()
		w.writeByte('}')
	}
}

func (w *Writer) writeTSTypePair(pair jsir.TSTypePair) {
	w.writeString(pair.Key)
	if pair.Method {
		w.writeTSFunction(pair.Value.(*jsir.TSFunction), true)
		return
	}
	if pair.Optional {
		w.writeByte('?')
	}
	w.writeString(": ")
	w.writeTSType(pair.Value)
}

// TypeScript Statements
// ==========

func (w *Writer) writeTSModifierStmt(stmt *jsir.TSModifierStatement) {
	if stmt.Modifier == 0 {
		panic("jsir.TSModifierStatement has no modifiers")
	}
	if (stmt.Modifier & jsir.TSDeclare) != 0 {
		w.writeString("declare ")
	}
	w.writeStatement(stmt)
}

func (w *Writer) writeTSGenericDecl(generics []jsir.TSGenericDecl) {
	w.writeByte('<')
	for i, gen := range generics {
		if i > 0 {
			w.writeString(", ")
		}
		w.writeString(gen.Name)
		if gen.Extends != nil {
			w.writeString(" extends ")
			w.writeTSType(gen.Extends)
		}
		if gen.Default != nil {
			w.writeString(" = ")
			w.writeTSType(gen.Default)
		}
	}
	w.writeByte('>')
}

func (w *Writer) writeTSInterfaceDecl(stmt *jsir.TSInterfaceDeclaration) {
	w.writeString("interface ")
	w.writeString(stmt.Name)
	if stmt.Generics != nil {
		w.writeTSGenericDecl(*stmt.Generics)
	}
	if stmt.Extends != nil {
		w.writeString(" extends ")
		for i, typ := range *stmt.Extends {
			if i > 0 {
				w.writeString(", ")
			}
			w.writeTSType(typ)
		}
	}
	w.writeByte(' ')
	w.writeTSObjectLiteral(&stmt.Value)
}
