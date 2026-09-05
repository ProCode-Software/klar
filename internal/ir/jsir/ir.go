package jsir

import "github.com/ProCode-Software/klar/internal/lexer"

type Module struct {
	Statements []Statement
	Comments   []Comment
}

type Comment struct {
	Position lexer.Position
	Text     string
	Kind     CommentKind
}

type CommentKind uint8

const (
	LineComment CommentKind = iota
	BlockComment
	Hashbang
)
