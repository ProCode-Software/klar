package lsp

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/module"
)

type Package struct {
	*module.PackageInfo
	Compiler *build.ProjectCompiler
}

type Module struct {
	*analysis.Module
}
