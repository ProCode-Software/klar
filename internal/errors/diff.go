package errors

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type DiffEdit interface {
	Operation() bool
	FullLine() bool
	Start() lexer.Position
	EndLine() uint32
}

type DeletedRange struct {
	Range ranges.Range
	Line  bool
}

func (r DeletedRange) Operation() bool       { return false }
func (r DeletedRange) FullLine() bool        { return r.Line }
func (r DeletedRange) Start() lexer.Position { return r.Range.Start }
func (r DeletedRange) EndLine() uint32       { return r.Range.End.Line }

type AddedTokens struct {
	Tokens   []lexer.Token
	Line     bool
}

func (r AddedTokens) Operation() bool       { return true }
func (r AddedTokens) FullLine() bool        { return r.Line }
func (r AddedTokens) Start() lexer.Position { return r.Tokens[0].Position }
func (r AddedTokens) EndLine() uint32 {
	return r.Tokens[len(r.Tokens)-1].Position.Line
}

type AddedString struct {
	Position lexer.Position
	String   string
	Line     bool
	NumLines uint32
}

func (r AddedString) Operation() bool       { return true }
func (r AddedString) FullLine() bool        { return r.Line }
func (r AddedString) Start() lexer.Position { return r.Position }
func (r AddedString) EndLine() uint32 {
	return r.Position.Line + max(1, r.NumLines) - 1
}
