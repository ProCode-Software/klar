package jsir

func (*IfStatement) _stmt()     {}
func (*SwitchStatement) _stmt() {}

func (*DebuggerStatement) _stmt()       {}
func (*ContinueStatement) _stmt()       {}
func (*BreakStatement) _stmt()          {}
func (*ThrowStatement) _stmt()          {}
func (*ReturnStatement) _stmt()         {}
func (*BindingDeclaration) _stmt()      {}
func (*MultiBindingDeclaration) _stmt() {}
func (*FunctionDeclaration) _stmt()     {}
func (*ClassDeclaration) _stmt()        {}
func (*AssignmentStatement) _stmt()     {}
func (*ExpressionStatement) _stmt()     {}
func (*ExportModifier) _stmt()          {}

func (*BinaryExpression) _expr() {}
func (*UnaryExpression) _expr()  {}
func (*NullLiteral) _expr()      {}
func (*UndefinedLiteral) _expr() {}
func (*Symbol) _expr()           {}
func (*BooleanLiteral) _expr()   {}
func (*NumericLiteral) _expr()   {}
func (*StringLiteral) _expr()    {}
func (*TemplateLiteral) _expr()  {}

// Functions and classes are also expressions in JS, even with names
func (*FunctionDeclaration) _expr() {}
func (*ClassDeclaration) _expr()    {}

func (*SpreadExpression) _expr() {}
func (*ArrayLiteral) _expr()     {}
func (*ObjectLiteral) _expr()    {}
