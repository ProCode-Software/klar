package main

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/paths"
)

var Colors ansi.Colors

func argAt(i int) (string, bool) {
	args := os.Args
	if i >= len(args) {
		return "", false
	}
	return args[i], true
}

func handleErr(err error, format ...string) {
	if err != nil {
		if len(format) == 0 {
			cli.InternalError(err)
		}
		cli.InternalError(fmt.Errorf(format[0], err))
	}
}

func ResolveManifest(path string) (manifest string) {
	manifest, found, err := module.ResolveManifest(path)
	if !found {
		cli.Failure("Project not found: ", fmt.Sprintf("Unable to find %s in %s",
			Colors.Yellow("glas.pack"),
			Colors.Cyan(paths.Full(path)),
		))
	}
	if err != nil {
		cli.InternalError(fmt.Errorf("Unable to resolve manifest from %s: %w", path, err))
	}
	return manifest
}
