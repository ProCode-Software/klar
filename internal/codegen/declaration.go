package codegen

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertObject(o *analysis.Object) jsir.Statement {
	switch stmt := o.Node().(type) {
	case *ast.FunctionDeclaration:
		return g.convertFuncDecl(o, stmt)
	case *ast.FuncAliasDeclaration:
		return g.convertFuncAlias(o, stmt)
	case *ast.VariableDeclaration:
		return g.convertVarDecl(o, stmt)
	case ast.TypeDeclaration:
		return g.convertTypeDecl(o, stmt)
	default:
		panic(fmt.Sprintf("invalid analysis.Object statement: %T", stmt))
	}
}

func (g *Generator) convertTypeDecl(
	o *analysis.Object, decl ast.TypeDeclaration,
) jsir.Statement {
	typ := o.TypeName().Type
	switch decl := decl.(type) {
	case *ast.StructDeclaration:
		return g.convertStruct(o, typ.(*analysis.Struct))
	case *ast.EnumDeclaration:
		return g.convertEnum(o, typ.(*analysis.Enum))
	case *ast.InterfaceDeclaration:
		return nil // TODO: Return one for TypeScript
	case *ast.TagDeclaration:
		return g.convertTagDecl(o)
	case *ast.TypeAliasDeclaration:
	default:
		panic(fmt.Sprintf("invalid ast.TypeDeclaration node: %T", decl))
	}
	return &jsir.FunctionDeclaration{
		Name: o.Name,
		Body: &jsir.Block{},
	}
}

func (g *Generator) convertVarDecl(
	o *analysis.Object, decl *ast.VariableDeclaration,
) jsir.Statement {
	// Use 'let' for all declarations unless they are public, top-level constants
	if len(decl.Variables) == 1 {
		// Convert `_ := expr` to `void expr`
		// TODO: [DCE] Don't generate anything at all if the value has no side effects
		if _, ok := decl.Variables[0].(*ast.Discard); ok {
			return &jsir.ExpressionStatement{&jsir.UnaryExpression{
				Operator: jsir.OpVoid,
				Operand:  g.convertExpression(decl.Values[0]),
			}}
		}
		return &jsir.BindingDeclaration{
			Kind: jsir.LetBinding,
			// Destructurability validated at analysis time
			Variable: g.convertDestructure(decl.Variables[0].(ast.Destructurable)),
			Value:    g.convertExpression(decl.Values[0]),
		}
	}

	jsDecl := &jsir.MultiBindingDeclaration{
		Kind:     jsir.LetBinding,
		Bindings: make([]*jsir.Binding, 0, len(decl.Variables)),
	}
	for i, name := range decl.Variables {
		// Don't declare discards. Remember that '_' is a valid identifier in JS.
		if _, ok := name.(*ast.Discard); ok {
			continue
		}
		var val ast.Expression
		if decl.IsSingleRHS() {
			val = decl.Values[0]
		} else {
			val = decl.Values[i]
		}
		jsDecl.Bindings = append(jsDecl.Bindings, &jsir.Binding{
			Variable: g.convertDestructure(name.(ast.Destructurable)),
			Value:    g.convertExpression(val),
		})
	}
	if len(jsDecl.Bindings) == 0 {
		// All variables declared are discards, this just becomes a 'void' expression
		return &jsir.ExpressionStatement{&jsir.UnaryExpression{
			Operator: jsir.OpVoid,
			Operand:  g.convertExpression(decl.Values[0]),
		}}
	}
	return jsDecl
}

func (g *Generator) convertFuncAlias(
	o *analysis.Object, decl *ast.FuncAliasDeclaration,
) *jsir.BindingDeclaration {
	return &jsir.BindingDeclaration{
		Kind:     jsir.ConstBinding,
		Variable: &jsir.Symbol{o.Name},
		Value:    &jsir.NullLiteral{},
	}
}

func (g *Generator) convertFuncDecl(
	o *analysis.Object, decl *ast.FunctionDeclaration,
) *jsir.FunctionDeclaration {
	// The self type will be disregarded as the generated function is put
	// into the JS class body
	return &jsir.FunctionDeclaration{
		Name: o.Name,
		Body: &jsir.Block{},
	}
}

func (g *Generator) convertTagDecl(o *analysis.Object) *jsir.BindingDeclaration {
	keyword := jsir.LetBinding
	if o.Public {
		keyword = jsir.ConstBinding
	}
	// Tags are converted to unique Symbols that are set as properties
	// on implementing classes
	return &jsir.BindingDeclaration{
		Kind:     keyword,
		Variable: &jsir.Symbol{o.Name},
		Value: &jsir.CallExpression{
			g.getGlobal("Symbol", o.Context), []jsir.Expression{
				// Use single quotes because the name is a valid identifier
				&jsir.StringLiteral{jsir.SingleQuote, o.Name},
			},
		},
	}
}

func (g *Generator) getGlobal(name string, ctx *analysis.Context) *jsir.Symbol {
	return &jsir.Symbol{Name: name}
}
