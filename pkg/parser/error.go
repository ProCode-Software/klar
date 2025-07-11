package parser

import "github.com/ProCode-Software/klar/internal/errors"

func IsKlarError(err error) bool {
	_, ok := err.(errors.KlarError)
	return ok
}

