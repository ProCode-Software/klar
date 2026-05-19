package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type expectFlag interface{ expect() }

var noAdvance _flagNoAdvance

type (
	_flagNoAdvance       struct{} // Don't advance when there is an error
	expectErrorCode errors.ErrorCode
	expectError     struct{ err *errors.ParseError } // [Parser.Expect] can modify this error
	withLabel       string
	withMessage     string
)

func (_flagNoAdvance) expect()       {}
func (expectErrorCode) expect() {}
func (expectError) expect()     {}
func (withLabel) expect()       {}
func (withMessage) expect()     {}

func withExpectFlags(flags []expectFlag, exp lexer.TokenType, got lexer.Token) (err *ParseError, stay bool) {
	for _, flag := range flags {
		switch flag := flag.(type) {
		case expectError:
			err = flag.err
			err.Token = got
			err.Range = ranges.FromToken(got)
			if err.ErrorCode == 0 {
				err.ErrorCode = errors.ErrExpectedToken
			}
		case withLabel:
			if err == nil {
				err = errors.Token(errors.ErrExpectedToken, got)
			}
			err.Label = string(flag)
		case withMessage:
			err = errors.ExpectedTokenf(string(flag), exp, got)
			err.SetParam("msg", string(flag))
		case expectErrorCode:
			err = errors.Token(errors.ErrorCode(flag), got)
		case _flagNoAdvance:
			stay = true
		}
	}
	err.SetParam("expected", exp)
	return
}
