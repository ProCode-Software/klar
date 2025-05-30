package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
)

type ParseError = errors.ParseError

type Settings struct {
	ContinueOnError bool
	OnError         func(e ParseError)
}

type ParseOptions struct {
	ContinueOnError bool
	OnError         func(e ParseError)
}
