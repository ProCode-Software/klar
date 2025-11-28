package reader

import (
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
)

const (
	_ errors.ErrorCode = iota
	ErrUnterminatedString
	ErrExpectedEOF
)

type ParseError struct {
	Code  errors.ErrorCode
	Range ranges.Range
	Token Token
}

func (err *ParseError) Error() string {
	return ""
}
