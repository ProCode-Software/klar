package codegen

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertObject(o *analysis.Object) jsir.Statement {
	switch stmt := o.Node().(type) {
	case *ast.FunctionDeclaration:
		return g.convertFuncDecl(o)
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
			Variable: g.convertDestructure(
				decl.Variables[0].(ast.Destructurable), !o.Public,
			),
			Value: g.convertExpression(decl.Values[0]),
		}
	}

	jsDecl := &jsir.MultiBindingDeclaration{
		Kind:     jsir.LetBinding,
		Bindings: make([]*jsir.Binding, 0, len(decl.Variables)),
	}
	var singleRHS jsir.Expression
	if decl.IsSingleRHS() {
		singleRHS = g.convertExpression(decl.Values[0])
	}
	for i, name := range decl.Variables {
		// Don't declare discards. Remember that '_' is a valid identifier in JS.
		if _, ok := name.(*ast.Discard); ok {
			continue
		}
		var val jsir.Expression
		if singleRHS != nil {
			val = singleRHS
		} else {
			val = g.convertExpression(decl.Values[i])
		}
		jsDecl.Bindings = append(jsDecl.Bindings, &jsir.Binding{
			Variable: g.convertDestructure(name.(ast.Destructurable), !o.Public),
			Value:    val,
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
		Variable: escapedSymbol(o.Name, !o.Public),
		Value:    &jsir.NullLiteral{},
	}
}

func (g *Generator) convertFuncDecl(o *analysis.Object) jsir.Statement {
	// Some overloads may have different names via the @name attribute.
	// Group the overloads by name and convert those grouped ones to JavaScript.
	var nameGroups map[string][]*analysis.Overload
	fnType := o.Type.(*analysis.Function)
	for _, ov := range fnType.Overloads {
		if ov.Name != o.Name {
			if nameGroups == nil {
				nameGroups = make(map[string][]*analysis.Overload)
			}
			nameGroups[ov.Name] = append(nameGroups[ov.Name], ov)
		}
	}
	var converted []jsir.Statement
	if nameGroups == nil {
		// No name changes
		converted = make([]jsir.Statement, 0, len(fnType.Overloads)+1)
		converted = g.appendOverloads(converted, fnType.Overloads)
	} else {
		// Some may still have the original name, so add that to nameGroups
		for _, fnType := range fnType.Overloads {
			if fnType.Name == o.Name {
				nameGroups[fnType.Name] = append(nameGroups[fnType.Name], fnType)
			}
		}
		converted = make([]jsir.Statement, 0, len(nameGroups)+len(fnType.Overloads))
		// Sort the overloads by declaration order
		sortOverloads := createOverloadSorter(nameGroups)
		for _, name := range slices.SortedFunc(maps.Keys(nameGroups), sortOverloads) {
			converted = g.appendOverloads(converted, nameGroups[name])
		}
	}
	if len(converted) == 1 { // Optimization when there's only 1 overload
		return converted[0]
	}
	return &jsir.StatementChain{converted}
}

func createOverloadSorter(nameGroups map[string][]*analysis.Overload) func(string, string) int {
	return func(a, b string) int {
		return int(nameGroups[a][0].Order) - int(nameGroups[b][0].Order)
	}
}

func (g *Generator) convertTagDecl(o *analysis.Object) *jsir.BindingDeclaration {
	keyword := jsir.LetBinding
	if o.Public {
		keyword = jsir.ConstBinding
	}
	// Tags are converted to unique Symbols that are set as properties
	// on implementing classes
	// 	const Node = Symbol('Node')
	return &jsir.BindingDeclaration{
		Kind:     keyword,
		Variable: escapedSymbol(o.Name, !o.Public),
		Value: &jsir.CallExpression{
			g.getGlobal("Symbol", o.Context), []jsir.Expression{
				// Use single quotes because the name is a valid identifier
				&jsir.StringLiteral{jsir.SingleQuote, o.Name},
			},
		},
	}
}

func (g *Generator) getGlobal(name string, ctx *analysis.Context) jsir.Expression {
	if ctx.LookupRecursive(name) != nil {
		return &jsir.IndexExpression{
			Object:   &jsir.Symbol{"globalThis"},
			Property: &jsir.Symbol{Name: name},
		}
	}
	return &jsir.Symbol{Name: name}
}
