package lexer

var TokenTypes = map[TokenType]string{
	EOF:            "EOF",
	EndOfStatement: "EndOfStatement",
	Illegal:        "Illegal",
	Newline:        "Newline",

	Comma:              "Comma",
	Dot:                "Dot",
	LineComment:        "LineComment",
	BlockComment:       "BlockComment",
	Colon:              "Colon",
	LeftBracket:        "LeftBracket",
	RightBracket:       "RightBracket",
	LeftParenthesis:    "LeftParenthesis",
	RightParenthesis:   "RightParenthesis",
	LeftCurlyBrace:     "LeftCurlyBrace",
	HashLeftCurlyBrace: "HashLeftCurlyBrace",
	RightCurlyBrace:    "RightCurlyBrace",
	At:                 "At",
	Identifier:         "Identifier",
	Numeric:            "Numeric",
	Boolean:            "Boolean",
	Nil:                "Nil",
	String:             "String",
	Discard:            "Discard",

	Plus:     "Plus",
	Minus:    "Minus",
	Times:    "Times",
	Divide:   "Divide",
	Modulo:   "Modulo",
	Exponent: "Exponent",

	EqualSign:  "EqualSign",
	ColonEqual: "ColonEqual",
	PlusEqual:  "PlusEqual",
	MinusEqual: "MinusEqual",
	Increment:  "Increment",
	Decrement:  "Decrement",

	Equals:         "Equals",
	NotEqual:       "NotEqual",
	GreaterThan:    "GreaterThan",
	LessThan:       "LessThan",
	GreaterEqualTo: "GreaterEqualTo",
	LessEqualTo:    "LessEqualTo",
	LogicalAnd:     "LogicalAnd",
	LogicalOr:      "LogicalOr",
	LogicalNot:     "LogicalNot",

	Alternative: "Alternative",
	Optional:    "Optional",

	Spread:   "Spread",
	Arrow:    "Arrow",
	Pipeline: "Pipeline",

	For:    "For",
	Func:   "Func",
	Import: "Import",
	Next:   "Next",
	Return: "Return",
	Type:   "Type",
	When:   "When",
}

func (t TokenType) String() string {
	return TokenTypes[t]
}
