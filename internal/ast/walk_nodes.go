package ast

func (BadExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Program) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ExpressionStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StringLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (IntegerLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (BooleanLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (NilLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (FloatLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (RegexLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (VersionLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Comment) Walk(Visitor, *Cursor) StopCode { return 0 }

func (BinaryExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (UnaryExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (RelationalExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Symbol) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Discard) Walk(Visitor, *Cursor) StopCode { return 0 }

func (PublicDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (VariableDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AssignmentStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ExpressionPair) Walk(Visitor, *Cursor) StopCode { return 0 }

func (PrimitiveType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TypeAlias) Walk(Visitor, *Cursor) StopCode { return 0 }

func (QualifiedTypeAlias) Walk(Visitor, *Cursor) StopCode { return 0 }

func (OptionalType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ListType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MapType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (RestType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TupleType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TypePair) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AssignableTuple) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AssignableTypePair) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ParenType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (FunctionType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (GenericType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (InterfaceItem) Walk(Visitor, *Cursor) StopCode { return 0 }

func (UnionType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MethodType) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MethodParam) Walk(Visitor, *Cursor) StopCode { return 0 }

func (IdentifierPair) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ImportStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (InterfaceDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TagDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StructDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StructField) Walk(Visitor, *Cursor) StopCode { return 0 }

func (EnumDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (EnumItem) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TypeAliasDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MapLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MapItem) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TupleLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ReturnStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Block) Walk(Visitor, *Cursor) StopCode { return 0 }

func (FunctionDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (FuncAliasDeclaration) Walk(Visitor, *Cursor) StopCode { return 0 }

func (FunctionParam) Walk(Visitor, *Cursor) StopCode { return 0 }

func (NextStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StopStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ListLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (IndexExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (SliceExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (EnumLiteral) Walk(Visitor, *Cursor) StopCode { return 0 }

func (CallParam) Walk(Visitor, *Cursor) StopCode { return 0 }

func (CallExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StructDotInit) Walk(Visitor, *Cursor) StopCode { return 0 }

func (UpdateStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ForStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (WhileStatement) Walk(Visitor, *Cursor) StopCode { return 0 }

func (WhenExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (WhenCase) Walk(Visitor, *Cursor) StopCode { return 0 }

func (WhenCanCase) Walk(Visitor, *Cursor) StopCode { return 0 }

func (LambdaExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (Attribute) Walk(Visitor, *Cursor) StopCode { return 0 }

func (RestExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (RangeExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (PipelineExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ParenExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ListCastExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (MapCastExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ForExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ObjectPipeline) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ListDestructure) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ObjectDestructure) Walk(Visitor, *Cursor) StopCode { return 0 }

func (ObjectDestructureEntry) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AssignableVars) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AwaitExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (GoExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (TryExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (AssertExpression) Walk(Visitor, *Cursor) StopCode { return 0 }

func (StringTypeMatch) Walk(Visitor, *Cursor) StopCode { return 0 }
