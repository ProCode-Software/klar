package parser

import (
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type expectFlag interface{ expect() }

var noAdvance _flagNoAdvance

type (
	_flagNoAdvance  struct{} // Don't advance when there is an error
	expectErrorCode klarerrs.Code
	expectError     struct{ err *klarerrs.Error } // [Parser.Expect] can modify this error
	withLabel       string
	withMessage     string
)

func (_flagNoAdvance) expect()  {}
func (expectErrorCode) expect() {}
func (expectError) expect()     {}
func (withLabel) expect()       {}
func (withMessage) expect()     {}

func withExpectFlags(
	flags []expectFlag, exp lexer.TokenType, got lexer.Token,
) (err *Error, stay bool) {
	for _, flag := range flags {
		switch flag := flag.(type) {
		case expectError:
			err = flag.err
			err.Info = klarerrs.TokenInfo(got)
			err.Range = ranges.FromToken(got)
			if err.Code == 0 {
				err.Code = klarerrs.ErrExpectedToken
			}
		case withLabel:
			if err == nil {
				err = klarerrs.Token(klarerrs.ErrExpectedToken, got)
			}
			err.Label = string(flag)
		case withMessage:
			err = klarerrs.ExpectedTokenf(string(flag), exp, got)
			err.SetParam("msg", string(flag))
		case expectErrorCode:
			err = klarerrs.Token(klarerrs.Code(flag), got)
		case _flagNoAdvance:
			stay = true
		}
	}
	if err == nil {
		err = klarerrs.Token(klarerrs.ErrExpectedToken, got)
	}
	err.SetParam("expected", exp)
	return
}
