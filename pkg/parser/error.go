package parser

import "github.com/ProCode-Software/klar/internal/errors"

func IsKlarError(err error) bool {
	_, is := err.(errors.KlarError)
	return is
}

func IsParseError(err error) bool {
	if !IsKlarError(err) {
		return false
	}
	_, is := err.(errors.ParseError)
	return is
}
