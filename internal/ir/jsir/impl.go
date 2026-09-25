package jsir

func (*StatementChain) _stmt()  {}
func (*IfStatement) _stmt()     {}
func (*SwitchStatement) _stmt() {}

func (*DebuggerStatement) _stmt()       {}
func (*ContinueStatement) _stmt()       {}
func (*BreakStatement) _stmt()          {}
func (*ThrowStatement) _stmt()          {}
func (*ReturnStatement) _stmt()         {}
func (*BindingDeclaration) _stmt()      {}
func (*MultiBindingDeclaration) _stmt() {}
func (*ExpressionStatement) _stmt()     {}
func (*ForStatement) _stmt()            {}
func (*WhileStatement) _stmt()          {}
func (*DoWhileStatement) _stmt()        {}
func (*TryStatement) _stmt()            {}
func (*EmptyStatement) _stmt()          {}
func (*LabelledStatement) _stmt()       {}
func (*Block) _stmt()                   {}
func (*ImportStatement) _stmt()         {}
func (*ExportModifierStatement) _stmt() {}
func (*NamedExportsStatement) _stmt()   {}
func (*ExportFromStatement) _stmt()     {}

func (*BinaryExpression) _expr()       {}
func (*UnaryExpression) _expr()        {}
func (*PostfixUnaryExpression) _expr() {}
func (*NullLiteral) _expr()            {}
func (*UndefinedLiteral) _expr()       {}
func (*Symbol) _expr()                 {}
func (*BooleanLiteral) _expr()         {}
func (*NumericLiteral) _expr()         {}
func (*RegExpLiteral) _expr()          {}
func (*StringLiteral) _expr()          {}
func (*TemplateLiteral) _expr()        {}
func (*AssignmentExpression) _expr()   {}
func (*SpreadExpression) _expr()       {}
func (*ArrayLiteral) _expr()           {}
func (*ObjectLiteral) _expr()          {}
func (*MemberExpression) _expr()       {}
func (*CallExpression) _expr()         {}
func (*TernaryExpression) _expr()      {}
func (*CommaExpression) _expr()        {}
func (*ArrowFunction) _expr()          {}

// Functions and classes are also expressions in JS, even with names
func (*FunctionDeclaration) _expr() {}
func (*ClassDeclaration) _expr()    {}
func (*FunctionDeclaration) _stmt() {}
func (*ClassDeclaration) _stmt()    {}

func (*Symbol) _tsType()           {}
func (*StringLiteral) _tsType()    {}
func (*BooleanLiteral) _tsType()   {}
func (*NumericLiteral) _tsType()   {}
func (*NullLiteral) _tsType()      {}
func (*UndefinedLiteral) _tsType() {}
func (*TSArray) _tsType()          {}
func (*TSComputedIndex) _tsType()  {}
func (*TSGeneric) _tsType()        {}
func (*TSImport) _tsType()         {}
func (*TSNamespaceIndex) _tsType() {}
func (*TSTuple) _tsType()          {}
func (*TSUnion) _tsType()          {}
func (*TSIntersection) _tsType()   {}
func (*TSObjectLiteral) _tsType()  {}
func (*TSFunction) _tsType()       {}

func (*TSModifierStatement) _stmt()    {}
func (*TSTypeDeclaration) _stmt()      {}
func (*TSInterfaceDeclaration) _stmt() {}
func (*TSNamespaceDeclaration) _stmt() {}
