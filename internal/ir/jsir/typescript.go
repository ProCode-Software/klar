package jsir

// Unlike the rest of the JavaScript IR, the format of TypeScript IR will be specific
// to Klar's needs.

type TSType interface{ _tsType() }

// TypeScript Statements
// =======

type TSModifier uint8

const (
	TSDeclare TSModifier = 1 << iota
)

type TSModifierStatement struct {
	Modifier  TSModifier
	Statement Statement
}

type TSTypeDeclaration struct {
	Name     string
	Generics *[]TSGenericDecl
	Value    TSType
}

type TSInterfaceDeclaration struct {
	Name     string
	Generics *[]TSGenericDecl
	Extends  *[]TSType
	Value    TSObjectLiteral
}

type TSNamespaceDeclaration struct {
	Namespace string
	Body      []Statement
}

type TSGenericDecl struct {
	Name    string
	Extends TSType // Can be nil
	Default TSType // Can be nil
}

// Type Annotations
// =======

// Namespace.Object
type TSNamespaceIndex struct {
	Chain []string
}

// Type[Index]
type TSComputedIndex struct{ Type, Index TSType }

// ItemType[]
type TSArray struct{ ItemType TSType }

// [Item1, Item2, ...]
type TSTuple struct{ Items []TSType }

// import('...')
type TSImport struct{ Path StringLiteral }

// Type<Arguments...>
type TSGeneric struct {
	Type      TSType
	Arguments []TSType
}

// Type1 | Type2
type TSUnion struct{ Items []TSType }

// Type1 & Type2
type TSIntersection struct{ Items []TSType }

type TSObjectLiteral struct {
	Properties         []TSTypePair
	ComputedProperties *[][2]TSType // [_: Type]: Type
	// TypeScript also has call signatures, but Klar doesn't currently need them
	// 	{ (args): ret }
}

type TSTypePair struct {
	Key      string
	Optional bool // ?:
	Method   bool // If true, Value must be [TSFunction]
	Value    TSType
}

type TSFunction struct {
	Arguments []TSTypePair
	Return    TSType
}
