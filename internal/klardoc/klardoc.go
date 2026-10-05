package klardoc

import (
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
)

func GenerateModule(mod *analysis.Module, files map[string]*ast.Program) *Module {
	docMod := &Module{
		ImportPath:   mod.ImportPathString(),
		SourcePath:   mod.Path,
		Files:        slices.Sorted(maps.Keys(files)),
		Declarations: make([]*Declaration, 0, len(mod.Context.Declarations)),
	}
	return docMod
}

// TODO: GeneratePackage and GenerateProject

func GeneratePackage() *Package {
	return nil
}

func GenerateProject() *ProjectKlarDocJSON {
	return nil
}
