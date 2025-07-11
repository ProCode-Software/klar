package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
)

type ParseError = errors.ParseError

type ParseOptions struct {
	StopOnError bool
	OnError     func(e ParseError)
}
