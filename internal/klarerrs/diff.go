package klarerrs

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Diff struct {
	File  string
	Edits []DiffEdit
}

func NewDiff(file string, edits ...DiffEdit) *Diff {
	return &Diff{File: file, Edits: edits}
}

type DiffEdit interface {
	Operation() bool // True for additions, false for deletions
	FullLine() bool
	Start() lexer.Position
	EndLine() uint32
}

type DeletedRange struct{ Range ranges.Range }

func (r DeletedRange) Operation() bool       { return false }
func (r DeletedRange) FullLine() bool        { return false }
func (r DeletedRange) Start() lexer.Position { return r.Range.Start }
func (r DeletedRange) EndLine() uint32       { return r.Range.End.Line }

type DeletedLine struct{ Line uint32 }

func (r DeletedLine) Operation() bool       { return false }
func (r DeletedLine) FullLine() bool        { return true }
func (r DeletedLine) Start() lexer.Position { return lexer.Position{r.Line, 1} }
func (r DeletedLine) EndLine() uint32       { return r.Line }

type AddedTokens struct {
	Tokens []lexer.Token
	Line   bool
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
