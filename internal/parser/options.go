package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
)

type ParseError = errors.ParseError

type ParseOptions struct {
	File        string
	StopOnError bool
	OnError     func(e ParseError)
}
