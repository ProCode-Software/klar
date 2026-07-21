package clean

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func Run(c *command.Runner) {
	if err := module.LoadSystemDirs(); err != nil {
		cli.Failure("Failed to resolve cache directory:", err)
	}
	clearProjCache := c.Flag("project").Bool()
	clearGlobalCache := c.Flag("global").Bool()
	if !clearProjCache && !clearGlobalCache {
		cli.Failure(
			"'--project' and '--global' can't both be disabled, otherwise there's no work to do!",
		)
	}
	globalCacheSize, projCacheSize := ClearCache(clearGlobalCache, clearProjCache)

	// Display the total size of the cleared cache
	totalSize := projCacheSize + globalCacheSize
	if totalSize == 0 {
		ansi.TagPrintln("<** g!>Good job!</> There's nothing to clean")
		return
	}
	formattedSize := util.FormatSize(totalSize)
	switch {
	case totalSize <= 50*1000: // 50 KB
		formattedSize = "<c!>" + formattedSize + "</c!>"
	case totalSize <= 5*(1000*1000): // 5 MB
		formattedSize = "<y!>" + formattedSize + "</y!>"
	default: // Over 5 MB
		formattedSize = "<r!>" + formattedSize + "</r!>"
	}
	ansi.TagPrintfln(
		"<g!>Successfully cleaned <**>%s</**> of build cache!</g>", formattedSize,
	)
}

func ClearCache(clearGlobalCache, clearProjCache bool) (globalCacheSize, projCacheSize int64) {
	// Delete global cache
	if clearGlobalCache {
		if _, err := os.Stat(module.SystemDirs.Cache); err == nil {
			if globalCacheSize, err = deleteDir(module.SystemDirs.Cache); err != nil {
				cli.Failure("Failed to delete cache directory:", err)
			}
			_ = os.Mkdir(module.SystemDirs.Cache, 0o755)
		}
	}

	// Delete project cache
	if clearProjCache {
		cwd, err := os.Getwd()
		if err != nil {
			cli.Warn("Skipped deleting the project cache due to an error:", err)
			return
		}
		_, projDir := module.PackageRoot(cwd)
		projCache := filepath.Join(projDir, module.LocalDataDir, "cache")
		if _, err := os.Stat(projCache); err == nil {
			if projCacheSize, err = deleteDir(projCache); err != nil {
				cli.Failure("Failed to delete project cache directory:", err)
			}
			// We won't recreate the project cache folder
		}
	}
	return
}

const LongDescription = `When building a project, Klar caches each built module to avoid recompiling everything, including unmodified modules, every time you build, making following builds faster. However, this takes up disk space, and deleted modules may still remain in cache.

You can occasionally clean the cache by running 'klar clean' to delete all cached modules. This doesn't delete or modify any of your Klar files. Remember that caching modules avoids typechecking them when they aren't modified, so deleting the cache will require recompiling the module. After running 'klar clean', the next build will take longer, but after building once, the cache will be resaved and build times will be faster for the next build onwards.`

func deleteDir(dir string) (size int64, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return os.Remove(path)
	})
	return
}

var Flags = argparse.NewParser().
	BoolFlag("project", "Clear project cache (in '.klar/cache') and not global cache", true, "p").
	BoolFlag("global", "Clear global cache (for all the user's projects)", true, "g")
