package klarnew

import (
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

type flags uint8

const (
	initGitRepo flags = 1 << iota
	addTests
	localCache
)

type createInfo struct {
	// If provided, a subpackage will be created for the specified project path
	rootProject string
	pkgName     string
	exec        bool // False if a library is being created
	flags       flags
}

func Run(c *command.Runner) {}

var Flags = argparse.NewParser("[dir]").
	EnumFlag("type", "The type of package to create", "", map[string]any{
		"library": 1, "lib": 1,
		"executable": 2, "exec": 2, "cmd": 2, "command": 2,
	}, "executable", "t").
	StringFlag("name", "The name of the package", "", "", "n").
	BoolFlag("add-tests", "Add test templates to the project", false).
	BoolFlag("local-cache", "Add a '.klar' folder to the project to store installed packages and cache", false).
	BoolFlag("git", "Initialize a new Git repository for the project", false)

const LongDescription = ""
