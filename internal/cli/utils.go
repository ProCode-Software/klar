package cli

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/paths"
)

func ResolveManifest(path string) (manifest string) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			InternalError(fmt.Errorf("Unable to resolve manifest: %w", err))
		}
	}
	manifest, found, err := module.ResolveManifest(path)
	if !found {
		Failure("Project not found: ", fmt.Sprintf("Unable to find %s in %s",
			ansi.Yellow("glas.pack"),
			ansi.Cyan(paths.Full(path)),
		))
	}
	if err != nil {
		InternalError(fmt.Errorf("Unable to resolve manifest from %s: %w", path, err))
	}
	return manifest
}
