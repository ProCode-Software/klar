package klarerrs

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestUnusedErrorCodes(t *testing.T) {
	// Find the module root
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var root string
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			root = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("failed to load packages: %v", err)
	}

	var klarerrsPkg *packages.Package
	for _, pkg := range pkgs {
		// Match the package by its path suffix to ensure we have the right one
		if pkg.Name == "klarerrs" && strings.HasSuffix(pkg.PkgPath, "internal/klarerrs") {
			klarerrsPkg = pkg
			break
		}
	}

	if klarerrsPkg == nil {
		t.Fatal("could not find internal/klarerrs package")
	}

	// 1. Identify all error codes (constants of type Code starting with Err or Warn)
	codeTypeObj := klarerrsPkg.Types.Scope().Lookup("Code")
	if codeTypeObj == nil {
		t.Fatal("could not find Code type in klarerrs")
	}
	codeType := codeTypeObj.Type()

	errorCodes := make(map[types.Object]string)
	for _, name := range klarerrsPkg.Types.Scope().Names() {
		obj := klarerrsPkg.Types.Scope().Lookup(name)
		if _, ok := obj.(*types.Const); ok && types.Identical(obj.Type(), codeType) {
			if strings.HasPrefix(name, "Err") || strings.HasPrefix(name, "Warn") {
				errorCodes[obj] = name
			}
		}
	}

	// 2. Identify internal dependencies within klarerrs
	// This maps a symbol (e.g., a preset function) to the symbols it uses.
	symbolUses := make(map[types.Object]map[types.Object]bool)

	for _, file := range klarerrsPkg.Syntax {
		filename := klarerrsPkg.Fset.File(file.Pos()).Name()
		// Skip generated files and formatting logic that just switch on error codes
		if strings.HasSuffix(filename, "_string.go") ||
			strings.Contains(filename, "error_message_test.go") ||
			strings.Contains(filename, "format.go") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				from := klarerrsPkg.TypesInfo.Defs[fn.Name]
				if from == nil {
					return true
				}

				// Skip handle* functions as they are formatting sinks, not "usages"
				if strings.HasPrefix(fn.Name.Name, "handle") {
					return false
				}

				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					if id, ok := inner.(*ast.Ident); ok {
						if to := klarerrsPkg.TypesInfo.Uses[id]; to != nil {
							if to.Pkg() != nil && to.Pkg().Path() == klarerrsPkg.Types.Path() {
								if symbolUses[from] == nil {
									symbolUses[from] = make(map[types.Object]bool)
								}
								symbolUses[from][to] = true
							}
						}
					}
					return true
				})
				return false
			}
			return true
		})
	}

	// 3. Identify external usages
	usedExternally := make(map[types.Object]bool)
	for _, pkg := range pkgs {
		// Ignore klarerrs itself and its internal tests
		if pkg.PkgPath == klarerrsPkg.PkgPath || strings.HasSuffix(pkg.PkgPath, ".test") {
			continue
		}
		for _, id := range pkg.TypesInfo.Uses {
			if id != nil && id.Pkg() != nil && id.Pkg().Path() == klarerrsPkg.Types.Path() {
				usedExternally[id] = true
			}
		}
	}

	// 4. Determine reachability (propagate usage from external references)
	allUsed := make(map[types.Object]bool)
	var queue []types.Object
	for obj := range usedExternally {
		if !allUsed[obj] {
			allUsed[obj] = true
			queue = append(queue, obj)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for child := range symbolUses[curr] {
			if !allUsed[child] {
				allUsed[child] = true
				queue = append(queue, child)
			}
		}
	}

	var unused []string
	for obj, name := range errorCodes {
		if !allUsed[obj] {
			unused = append(unused, name)
		}
	}

	if len(unused) > 0 {
		t.Errorf("%d unused error codes found:\n  %s", len(unused), strings.Join(unused, "\n  "))
	}
}
