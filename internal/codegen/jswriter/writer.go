package jswriter

import (
	"io"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

type Writer struct {
	Output map[string]io.Writer
	Inputs map[string]*InputFile
}

type InputFile struct {
	*jsir.Module
	TypeData *analysis.Module
}
