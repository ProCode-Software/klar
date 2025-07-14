package main

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
)

func RunBuild() {
	cmd := cli.NewArgParser()
	cmd.Parse()
	projDir := cmd.ArgAt(1)
	if projDir == "" {
		cwd, err := os.Getwd()
		handleErr(err)
		projDir = cwd
	}
	manifest := ResolveManifest(projDir)
	fmt.Println(manifest)
}
