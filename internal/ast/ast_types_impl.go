package ast

// AST items
func (Program) Kind() string             { return "Program" }
func (StringLiteral) Kind() string       { return "StringLiteral" }
func (FloatLiteral) Kind() string        { return "FloatLiteral" }
func (IntegerLiteral) Kind() string      { return "IntegerLiteral" }
func (BooleanLiteral) Kind() string      { return "BooleanLiteral" }
func (NilLiteral) Kind() string          { return "NilLiteral" }
func (ExpressionStatement) Kind() string { return "ExpressionStatement" }
func (BinaryExpression) Kind() string    { return "BinaryExpression" }
func (VariableDeclaration) Kind() string { return "VariableDeclaration" }
func (AssignmentStatement) Kind() string { return "AssignmentStatement" }
func (UnaryExpression) Kind() string     { return "AssignmentStatement" }
func (Symbol) Kind() string              { return "Symbol" }
func (ImportStatement) Kind() string     { return "ImportStatement" }
func (TypeAnnotation) Kind() string      { return "typeAnnotation" }

// String escapes
func (CharacterEscape) StringEscape()     {}
func (UnicodeEscape) StringEscape()       {}
func (HexadecimalEscape) StringEscape()   {}
func (StringInterpolation) StringEscape() {}

// Expressions
func (BinaryExpression) Expression() {}
func (UnaryExpression) Expression()  {}
func (NilLiteral) Expression()       {}
func (StringLiteral) Expression()    {}
func (IntegerLiteral) Expression()   {}
func (FloatLiteral) Expression()     {}
func (BooleanLiteral) Expression()   {}
func (Symbol) Expression()           {}
func (TypeAnnotation) Expression()   {}

// Statement
func (VariableDeclaration) Statement() {}
func (AssignmentStatement) Statement() {}
func (ExpressionStatement) Statement() {}
func (ImportStatement) Statement()     {}

// Type
func (PrimitiveType) Type() {}
func (TypeAlias) Type()     {}
func (OptionalType) Type()  {}
func (ListType) Type()      {}
func (RestType) Type()      {}
func (TupleType) Type()     {}
func (InterfaceType) Type() {}
func (FunctionType) Type()  {}
func (GenericType) Type()   {}
func (UnionType) Type()     {}

// Simple type
// Interface types aren't simple types
func (PrimitiveType) SimpleType() {}
func (TypeAlias) SimpleType()     {}
func (OptionalType) SimpleType()  {}
func (ListType) SimpleType()      {}
func (RestType) SimpleType()      {}
func (TupleType) SimpleType()     {}
func (FunctionType) SimpleType()  {}
func (GenericType) SimpleType()   {}
func (UnionType) SimpleType()     {}
