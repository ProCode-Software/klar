package glaspack

// This file declares wrapper types for encoding/decoding from Klon.

type DependencyCoder struct {
	Name string
	DependencySpecifier
}

// DependencyCoder should handle classes
