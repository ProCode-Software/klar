package glaspack

import "github.com/ProCode-Software/klar/pkg/klon/ast"

// This file declares wrapper types for encoding/decoding from Klon.

type DependencyCoder struct{ DependencySpecifier }

// DependencyCoder should handle classes
func (dc *DependencyCoder) UnmarshallKlon(val ast.Value) error {
	return nil
}
