package main

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/version"
)

var _helpString =
`Klar v%s

Usage: klar <command> [flags]

Commands:
	build		Build a Klar project
`

var HelpString = fmt.Sprintf(_helpString, version.KlarVersion)