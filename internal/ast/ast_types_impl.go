package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Base
func (node *BaseNode) SetPos(start, end lexer.Position) {
	node.Range.Start = start
	node.Range.End = end
}
func (node *BaseNode) GetRange() ranges.Range { return node.Range }

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
func (RegexLiteral) expr()       {}
func (VersionLiteral) expr()     {}
func (ListCastExpression) expr() {}
func (ObjectPipeline) expr()     {}
func (ForExpression) expr()      {}

// Statement
func (BadExpression) stmt()        {}
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
func (BreakStatement) stmt()       {}
func (FunctionDeclaration) stmt()  {}
func (NextStatement) stmt()        {}
func (Attribute) stmt()            {}
func (FuncAliasDeclaration) stmt() {}
func (PublicDeclaration) stmt()    {}

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

// Destructuring
func (ListLiteral) assignable() {}
func (TypeTuple) assignable()   {}
