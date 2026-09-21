package jsir

type Module struct {
	Statements []Statement
	Comments   []Comment
	Decls      map[string]Statement // Top-level declarations
}

type NodeAnchor int

type Offset struct {
	AnchorNode            NodeAnchor
	OffsetUp, OffsetRight uint32
}

type Comment struct {
	Offset Offset
	Text   string
	Kind   CommentKind
}

type CommentKind uint8

const (
	LineComment  CommentKind = iota // //
	BlockComment                    // /*
	Hashbang                        // #!
)

// This is to avoid allocating new [NullLiteral]s for each null value
var (
	Null      = &NullLiteral{}
	Undefined = &UndefinedLiteral{}
)

// See https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Lexical_grammar
var JSKeywords = map[string]struct{}{
	"arguments": {}, "async": {}, "await": {}, "break": {}, "case": {}, "catch": {},
	"class": {}, "const": {}, "continue": {}, "debugger": {}, "default": {}, "delete": {},
	"do": {}, "else": {}, "enum": {}, "eval": {}, "export": {}, "extends": {},
	"false": {}, "finally": {}, "for": {}, "function": {}, "if": {}, "implements": {},
	"import": {}, "in": {}, "instanceof": {}, "interface": {}, "let": {}, "new": {},
	"null": {}, "package": {}, "private": {}, "protected": {}, "public": {}, "return": {},
	"static": {}, "super": {}, "switch": {}, "this": {}, "throw": {}, "true": {},
	"try": {}, "typeof": {}, "undefined": {}, "var": {}, "void": {}, "while": {},
	"with": {}, "yield": {},
	// Added myself
	"using": {},
}