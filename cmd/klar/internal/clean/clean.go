package clean

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
)

func Run(c *command.Runner) {
	if err := module.LoadSystemDirs(); err != nil {
		cli.Failure("Failed to resolve cache directory:", err)
	}

	// Delete global cache
	if _, err := os.Stat(module.SystemDirs.Cache); err == nil {
		if err := os.RemoveAll(module.SystemDirs.Cache); err != nil {
			cli.Failure("Failed to delete cache directory:", err)
		}
		_ = os.Mkdir(module.SystemDirs.Cache, 0o755)
	}
	// Delete project cache
	cwd, err := os.Getwd()
	if err != nil {
		cli.Warn("Skipped deleting the project cache due to an error:", err)
		return
	}
	_, projDir := module.PackageRoot(cwd)
	projCache := filepath.Join(projDir, module.LocalDataDir, "cache")
	if _, err := os.Stat(projCache); err == nil {
		if err := os.RemoveAll(projCache); err != nil {
			cli.Failure("Failed to delete project cache directory:", err)
		}
		// We won't recreate the project cache folder
	}

	// TODO: Display the total size of the cache
	fmt.Println(ansi.BrightGreen("Successfully cleared the build cache!"))
}

const LongDescription = `When building a project, Klar caches each built module to avoid recompiling everything, including unmodified modules, every time you build, making following builds faster. However, this takes up disk space, and deleted modules may still remain in cache.

You can occasionally clean the cache by running 'klar clean' to delete all cached modules. This doesn't delete or modify any of your Klar files. Remember that caching modules avoids typechecking them when they aren't modified, so deleting the cache will require recompiling the module. After running 'klar clean', the next build will take longer, but after building once, the cache will be resaved and build times will be faster for the next build onwards.`
