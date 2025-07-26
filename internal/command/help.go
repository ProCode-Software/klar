package command

import "strings"

type HelpBuilder struct {
	Width int
	strings.Builder
}
