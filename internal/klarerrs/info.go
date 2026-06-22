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

type ModuleErrorInfo struct {
	ImportPath string
	Err        error
}

func (ModuleErrorInfo) info() {}
func (e *Error) ModuleErrorInfo() ModuleErrorInfo {
	return e.Info.(ModuleErrorInfo)
}

type TypeErrorInfo struct {
	ExpectedType, GotType string
}

func (TypeErrorInfo) info() {}
func (e *Error) TypeErrorInfo() TypeErrorInfo {
	return e.Info.(TypeErrorInfo)
}
