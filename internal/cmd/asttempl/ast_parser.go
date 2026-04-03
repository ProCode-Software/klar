package main

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/packages"
)

var (
	nodeIface               *types.Interface
	statementIface          *types.Interface
	expressionIface         *types.Interface
	assignableIface         *types.Interface
	typeIface               *types.Interface
	typeDeclarationIface    *types.Interface
	modifierDeclarationIface *types.Interface
	destructureIface         *types.Interface
)

// GetASTNodes returns all types in internal/ast that implement Node.
func GetASTNodes() ([]*types.TypeName, *types.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedImports | packages.NeedDeps,
	}

	pkgs, err := packages.Load(cfg, "github.com/ProCode-Software/klar/internal/ast")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load package: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		return nil, nil, fmt.Errorf("package has errors")
	}

	pkg := pkgs[0].Types
	scope := pkg.Scope()

	// Initialize interfaces
	if nodeIface = getInterface(scope, "Node"); nodeIface == nil {
		return nil, nil, fmt.Errorf("Node interface not found")
	}
	statementIface = getInterface(scope, "Statement")
	expressionIface = getInterface(scope, "Expression")
	assignableIface = getInterface(scope, "Assignable")
	typeIface = getInterface(scope, "Type")
	typeDeclarationIface = getInterface(scope, "TypeDeclaration")
	modifierDeclarationIface = getInterface(scope, "ModifierDeclaration")
	destructureIface = getInterface(scope, "Destructure")

	var nodes []*types.TypeName
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		typeName, ok := obj.(*types.TypeName)
		if !ok || !typeName.Exported() {
			continue
		}
		if _, ok := typeName.Type().Underlying().(*types.Interface); ok {
			continue
		}
		// Check if it's a struct and implements Node
		if IsNode(typeName.Type()) {
			nodes = append(nodes, typeName)
		}
	}

	return nodes, pkg, nil
}

func getInterface(scope *types.Scope, name string) *types.Interface {
	obj := scope.Lookup(name)
	if obj == nil {
		return nil
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	return iface
}

func IsNode(t types.Type) bool                { return implements(t, nodeIface) }
func IsStatement(t types.Type) bool           { return implements(t, statementIface) }
func IsExpression(t types.Type) bool          { return implements(t, expressionIface) }
func IsAssignable(t types.Type) bool          { return implements(t, assignableIface) }
func IsType(t types.Type) bool                { return implements(t, typeIface) }
func IsTypeDeclaration(t types.Type) bool     { return implements(t, typeDeclarationIface) }
func IsModifierDeclaration(t types.Type) bool { return implements(t, modifierDeclarationIface) }
func IsDestructure(t types.Type) bool         { return implements(t, destructureIface) }

func implements(t types.Type, iface *types.Interface) bool {
	if iface == nil || t == nil {
		return false
	}
	// Check if the type itself or its pointer version implements the interface.
	// Most AST nodes implement methods on pointer receivers.
	if types.Implements(t, iface) {
		return true
	}
	if _, ok := t.Underlying().(*types.Interface); ok {
		return types.Implements(t, iface)
	}
	return types.Implements(types.NewPointer(t), iface)
}
