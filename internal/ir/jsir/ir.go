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
