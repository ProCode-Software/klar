package ast

// AST items
func (Program) Kind() string              { return "Program" }
func (StringLiteral) Kind() string        { return "StringLiteral" }
func (FloatLiteral) Kind() string         { return "FloatLiteral" }
func (IntegerLiteral) Kind() string       { return "IntegerLiteral" }
func (BooleanLiteral) Kind() string       { return "BooleanLiteral" }
func (NilLiteral) Kind() string           { return "NilLiteral" }
func (ExpressionStatement) Kind() string  { return "ExpressionStatement" }
func (BinaryExpression) Kind() string     { return "BinaryExpression" }
func (VariableDeclaration) Kind() string  { return "VariableDeclaration" }
func (AssignmentStatement) Kind() string  { return "AssignmentStatement" }
func (UnaryExpression) Kind() string      { return "UnaryExpression" }
func (Symbol) Kind() string               { return "Symbol" }
func (ImportStatement) Kind() string      { return "ImportStatement" }
func (TypeAnnotation) Kind() string       { return "TypeAnnotation" }
func (EnumDeclaration) Kind() string      { return "EnumDeclaration" }
func (StructDeclaration) Kind() string    { return "StructDeclaration" }
func (TypeAliasDeclaration) Kind() string { return "TypeAliasDeclaration" }
func (MapLiteral) Kind() string           { return "MapLiteral" }
func (TupleLiteral) Kind() string         { return "TupleLiteral" }
func (ReturnStatement) Kind() string      { return "ReturnStatement" }
func (FunctionDeclaration) Kind() string  { return "FunctionDeclaration" }
func (NextStatement) Kind() string        { return "NextStatement" }
func (ListLiteral) Kind() string          { return "ListLiteral" }
func (IndexExpression) Kind() string      { return "IndexExpression" }
func (CallExpression) Kind() string       { return "CallExpression" }
func (EnumLiteral) Kind() string          { return "EnumValue" }
func (ForStatement) Kind() string         { return "ForStatement" }
func (UpdateStatement) Kind() string      { return "UpdateStatement" }
func (Discard) Kind() string              { return "Discard" }
func (WhenBlock) Kind() string            { return "WhenBlock" }
func (LambdaExpression) Kind() string     { return "LambdaExpression" }
func (ParamTuple) Kind() string           { return "ParamTuple" }
func (Attribute) Kind() string            { return "Attribute" }

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
func (MapLiteral) Expression()       {}
func (TupleLiteral) Expression()     {}
func (ListLiteral) Expression()      {}
func (IndexExpression) Expression()  {}
func (CallExpression) Expression()   {}
func (EnumLiteral) Expression()      {}
func (Discard) Expression()          {}
func (WhenBlock) Expression()        {}
func (ParamTuple) Expression()       {}
func (LambdaExpression) Expression() {}

// Statement
func (VariableDeclaration) Statement()  {}
func (UpdateStatement) Statement()      {}
func (ForStatement) Statement()         {}
func (AssignmentStatement) Statement()  {}
func (ExpressionStatement) Statement()  {}
func (ImportStatement) Statement()      {}
func (EnumDeclaration) Statement()      {}
func (StructDeclaration) Statement()    {}
func (TypeAliasDeclaration) Statement() {}
func (ReturnStatement) Statement()      {}
func (FunctionDeclaration) Statement()  {}
func (NextStatement) Statement()        {}
func (WhenBlock) Statement()            {}
func (Attribute) Statement()            {}

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

// Type declaration
func (TypeAliasDeclaration) TypeDeclaration() {}
func (StructDeclaration) TypeDeclaration()    {}
func (EnumDeclaration) TypeDeclaration()      {}

// Assignable types
func (Symbol) Assignable()          {}
func (IndexExpression) Assignable() {}
func (TupleLiteral) Assignable()    {}
