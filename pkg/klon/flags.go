package klon

import "github.com/ProCode-Software/klar/pkg/klon/klonflags"

func parseFlags(flags ...klonflags.Flags) klonflags.Flags {
	var f klonflags.Flags
	for _, flag := range flags {
		f |= flag
	}
	return f
}
