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
func (WhenExpression) Kind() string       { return "WhenExpression" }
func (LambdaExpression) Kind() string     { return "LambdaExpression" }
func (TypeTuple) Kind() string            { return "TypeTuple" }
func (Attribute) Kind() string            { return "Attribute" }
func (RangeExpression) Kind() string      { return "RangeExpression" }
func (RestExpression) Kind() string       { return "RestExpression" }
func (PipelineExpression) Kind() string   { return "PipelineExpression" }
func (BadExpression) Kind() string        { return "BadExpression" }
func (SliceExpression) Kind() string      { return "SliceExpression" }
func (ParenExpression) Kind() string      { return "ParenExpression" }
func (InterfaceDeclaration) Kind() string { return "InterfaceDeclaration" }
func (Comment) Kind() string              { return "Comment" }

// Implementations for types
func (PrimitiveType) Kind() string { return "PrimitiveType" }
func (TypeAlias) Kind() string     { return "TypeAlias" }
func (OptionalType) Kind() string  { return "OptionalType" }
func (ListType) Kind() string      { return "ListType" }
func (RestType) Kind() string      { return "RestType" }
func (TupleType) Kind() string     { return "TupleType" }
func (FunctionType) Kind() string  { return "FunctionType" }
func (GenericType) Kind() string   { return "GenericType" }
func (UnionType) Kind() string     { return "UnionType" }

// String escapes
func (BadEscape) StringEscape()           {}
func (CharacterEscape) StringEscape()     {}
func (UnicodeEscape) StringEscape()       {}
func (HexadecimalEscape) StringEscape()   {}
func (StringInterpolation) StringEscape() {}

// Expressions
func (BinaryExpression) Expression()   {}
func (UnaryExpression) Expression()    {}
func (NilLiteral) Expression()         {}
func (StringLiteral) Expression()      {}
func (IntegerLiteral) Expression()     {}
func (FloatLiteral) Expression()       {}
func (BooleanLiteral) Expression()     {}
func (Symbol) Expression()             {}
func (TypeAnnotation) Expression()     {}
func (MapLiteral) Expression()         {}
func (TupleLiteral) Expression()       {}
func (ListLiteral) Expression()        {}
func (IndexExpression) Expression()    {}
func (CallExpression) Expression()     {}
func (EnumLiteral) Expression()        {}
func (Discard) Expression()            {}
func (WhenExpression) Expression()     {}
func (TypeTuple) Expression()          {}
func (LambdaExpression) Expression()   {}
func (RangeExpression) Expression()    {}
func (RestExpression) Expression()     {}
func (PipelineExpression) Expression() {}
func (BadExpression) Expression()      {}
func (SliceExpression) Expression()    {}
func (ParenExpression) Expression()    {}

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
func (InterfaceDeclaration) Statement() {}
func (ReturnStatement) Statement()      {}
func (FunctionDeclaration) Statement()  {}
func (NextStatement) Statement()        {}
func (Attribute) Statement()            {}

// Type
func (PrimitiveType) Type() {}
func (TypeAlias) Type()     {}
func (OptionalType) Type()  {}
func (ListType) Type()      {}
func (RestType) Type()      {}
func (TupleType) Type()     {}
func (FunctionType) Type()  {}
func (GenericType) Type()   {}
func (UnionType) Type()     {}
func (BadExpression) Type() {}

// Type declaration
func (TypeAliasDeclaration) TypeDeclaration() {}
func (StructDeclaration) TypeDeclaration()    {}
func (EnumDeclaration) TypeDeclaration()      {}
func (InterfaceDeclaration) TypeDeclaration() {}

// Assignable types
func (Symbol) Assignable()          {}
func (IndexExpression) Assignable() {}
func (SliceExpression) Assignable() {}
func (TupleLiteral) Assignable()    {}
func (BadExpression) Assignable()   {}

// Publicizable declarations
func (d VariableDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d EnumDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d FunctionDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d StructDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d TypeAliasDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d InterfaceDeclaration) Publicize() Publicizable {
	d.Public = true
	return d
}

func (d VariableDeclaration) IsPublic() bool  { return d.Public }
func (d EnumDeclaration) IsPublic() bool      { return d.Public }
func (d FunctionDeclaration) IsPublic() bool  { return d.Public }
func (d StructDeclaration) IsPublic() bool    { return d.Public }
func (d TypeAliasDeclaration) IsPublic() bool { return d.Public }
func (d InterfaceDeclaration) IsPublic() bool { return d.Public }
