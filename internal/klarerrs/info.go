package klarerrs

import "github.com/ProCode-Software/klar/internal/lexer"

type Info interface {
	info()
}

type SyntaxErrorInfo struct {
	Token lexer.Token
}

func (SyntaxErrorInfo) info()      {}
func TokenInfo(t lexer.Token) Info { return SyntaxErrorInfo{Token: t} }
