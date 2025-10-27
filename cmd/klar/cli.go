package main

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

var Colors ansi.Colors

func handleErr(err error, format ...string) {
	if err != nil {
		if len(format) == 0 {
			cli.InternalError(err)
		}
		cli.InternalError(fmt.Errorf(format[0], err))
	}
}
