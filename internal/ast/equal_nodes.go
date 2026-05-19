package ast

func (BadExpression) Equal(Node) bool { return false }

func (Program) Equal(Node) bool { return false }

func (ExpressionStatement) Equal(Node) bool { return false }

func (StringLiteral) Equal(Node) bool { return false }

func (IntegerLiteral) Equal(Node) bool { return false }

func (BooleanLiteral) Equal(Node) bool { return false }

func (NilLiteral) Equal(Node) bool { return false }

func (FloatLiteral) Equal(Node) bool { return false }

func (RegexLiteral) Equal(Node) bool { return false }

func (VersionLiteral) Equal(Node) bool { return false }

func (Comment) Equal(Node) bool { return false }

func (BinaryExpression) Equal(Node) bool { return false }

func (UnaryExpression) Equal(Node) bool { return false }

func (RelationalExpression) Equal(Node) bool { return false }

func (Symbol) Equal(Node) bool { return false }

func (Discard) Equal(Node) bool { return false }

func (PublicDeclaration) Equal(Node) bool { return false }

func (VariableDeclaration) Equal(Node) bool { return false }

func (AssignmentStatement) Equal(Node) bool { return false }

func (ExpressionPair) Equal(Node) bool { return false }

func (PrimitiveType) Equal(Node) bool { return false }

func (TypeAlias) Equal(Node) bool { return false }

func (QualifiedTypeAlias) Equal(Node) bool { return false }

func (OptionalType) Equal(Node) bool { return false }

func (ListType) Equal(Node) bool { return false }

func (MapType) Equal(Node) bool { return false }

func (RestType) Equal(Node) bool { return false }

func (TupleType) Equal(Node) bool { return false }

func (TypePair) Equal(Node) bool { return false }

func (AssignableTuple) Equal(Node) bool { return false }

func (AssignableTypePair) Equal(Node) bool { return false }

func (ParenType) Equal(Node) bool { return false }

func (FunctionType) Equal(Node) bool { return false }

func (GenericType) Equal(Node) bool { return false }

func (InterfaceItem) Equal(Node) bool { return false }

func (UnionType) Equal(Node) bool { return false }

func (MethodType) Equal(Node) bool { return false }

func (MethodParam) Equal(Node) bool { return false }

func (IdentifierPair) Equal(Node) bool { return false }

func (ImportStatement) Equal(Node) bool { return false }

func (InterfaceDeclaration) Equal(Node) bool { return false }

func (TagDeclaration) Equal(Node) bool { return false }

func (StructDeclaration) Equal(Node) bool { return false }

func (StructField) Equal(Node) bool { return false }

func (EnumDeclaration) Equal(Node) bool { return false }

func (EnumItem) Equal(Node) bool { return false }

func (TypeAliasDeclaration) Equal(Node) bool { return false }

func (MapLiteral) Equal(Node) bool { return false }

func (MapItem) Equal(Node) bool { return false }

func (TupleLiteral) Equal(Node) bool { return false }

func (ReturnStatement) Equal(Node) bool { return false }

func (Block) Equal(Node) bool { return false }

func (FunctionDeclaration) Equal(Node) bool { return false }

func (FuncAliasDeclaration) Equal(Node) bool { return false }

func (FunctionParam) Equal(Node) bool { return false }

func (NextStatement) Equal(Node) bool { return false }

func (StopStatement) Equal(Node) bool { return false }

func (ListLiteral) Equal(Node) bool { return false }

func (IndexExpression) Equal(Node) bool { return false }

func (SliceExpression) Equal(Node) bool { return false }

func (EnumLiteral) Equal(Node) bool { return false }

func (CallParam) Equal(Node) bool { return false }

func (CallExpression) Equal(Node) bool { return false }

func (StructDotInit) Equal(Node) bool { return false }

func (UpdateStatement) Equal(Node) bool { return false }

func (ForStatement) Equal(Node) bool { return false }

func (WhileStatement) Equal(Node) bool { return false }

func (WhenExpression) Equal(Node) bool { return false }

func (WhenCase) Equal(Node) bool { return false }

func (WhenCanCase) Equal(Node) bool { return false }

func (LambdaExpression) Equal(Node) bool { return false }

func (Attribute) Equal(Node) bool { return false }

func (RestExpression) Equal(Node) bool { return false }

func (RangeExpression) Equal(Node) bool { return false }

func (PipelineExpression) Equal(Node) bool { return false }

func (ParenExpression) Equal(Node) bool { return false }

func (ListCastExpression) Equal(Node) bool { return false }

func (MapCastExpression) Equal(Node) bool { return false }

func (ForExpression) Equal(Node) bool { return false }

func (ObjectPipeline) Equal(Node) bool { return false }

func (ListDestructure) Equal(Node) bool { return false }

func (ObjectDestructure) Equal(Node) bool { return false }

func (ObjectDestructureEntry) Equal(Node) bool { return false }

func (AssignableVars) Equal(Node) bool { return false }

func (AwaitExpression) Equal(Node) bool { return false }

func (GoExpression) Equal(Node) bool { return false }

func (TryExpression) Equal(Node) bool { return false }

func (AssertExpression) Equal(Node) bool { return false }

func (StringTypeMatch) Equal(Node) bool { return false }
