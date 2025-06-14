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
func (MethodType) Kind() string           { return "MethodType" }

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
func (BadEscape) stringEsc()           {}
func (CharacterEscape) stringEsc()     {}
func (UnicodeEscape) stringEsc()       {}
func (HexadecimalEscape) stringEsc()   {}
func (StringInterpolation) stringEsc() {}

// Expressions
func (BinaryExpression) expr()   {}
func (UnaryExpression) expr()    {}
func (NilLiteral) expr()         {}
func (StringLiteral) expr()      {}
func (IntegerLiteral) expr()     {}
func (FloatLiteral) expr()       {}
func (BooleanLiteral) expr()     {}
func (Symbol) expr()             {}
func (TypeAnnotation) expr()     {}
func (MapLiteral) expr()         {}
func (TupleLiteral) expr()       {}
func (ListLiteral) expr()        {}
func (IndexExpression) expr()    {}
func (CallExpression) expr()     {}
func (EnumLiteral) expr()        {}
func (Discard) expr()            {}
func (WhenExpression) expr()     {}
func (TypeTuple) expr()          {}
func (LambdaExpression) expr()   {}
func (RangeExpression) expr()    {}
func (RestExpression) expr()     {}
func (PipelineExpression) expr() {}
func (BadExpression) expr()      {}
func (SliceExpression) expr()    {}
func (ParenExpression) expr()    {}

// Statement
func (VariableDeclaration) stmt()  {}
func (UpdateStatement) stmt()      {}
func (ForStatement) stmt()         {}
func (AssignmentStatement) stmt()  {}
func (ExpressionStatement) stmt()  {}
func (ImportStatement) stmt()      {}
func (EnumDeclaration) stmt()      {}
func (StructDeclaration) stmt()    {}
func (TypeAliasDeclaration) stmt() {}
func (InterfaceDeclaration) stmt() {}
func (ReturnStatement) stmt()      {}
func (FunctionDeclaration) stmt()  {}
func (NextStatement) stmt()        {}
func (Attribute) stmt()            {}

// Type
func (PrimitiveType) _type() {}
func (TypeAlias) _type()     {}
func (OptionalType) _type()  {}
func (ListType) _type()      {}
func (RestType) _type()      {}
func (TupleType) _type()     {}
func (FunctionType) _type()  {}
func (GenericType) _type()   {}
func (UnionType) _type()     {}
func (MethodType) _type()    {}
func (BadExpression) _type() {}

// Type declaration
func (TypeAliasDeclaration) typeDecl()      {}
func (StructDeclaration) typeDecl()         {}
func (EnumDeclaration) typeDecl()           {}
func (InterfaceDeclaration) typeDecl()      {}
func (d TypeAliasDeclaration) Name() string { return d.Identifier }
func (d StructDeclaration) Name() string    { return d.Identifier }
func (d EnumDeclaration) Name() string      { return d.Identifier }
func (d InterfaceDeclaration) Name() string { return d.Identifier }

// Assignable types
func (Symbol) assignable()          {}
func (IndexExpression) assignable() {}
func (SliceExpression) assignable() {}
func (TupleLiteral) assignable()    {}
func (BadExpression) assignable()   {}

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
