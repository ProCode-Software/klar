package cli

import "strings"

type HelpBuilder struct {
	MaxLength int
	strings.Builder
}

