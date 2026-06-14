package glascmd

var Aliases = map[string]string{
	// Alias -> Command
	"i":         "install",
	"a":         "add",
	"+":         "add",
	"u":         "remove",
	"uninstall": "remove",
	"r":         "remove",
	"ls":        "list",
	"p":         "publish",
	"pub":       "publish",
	"up":        "update",
}

// Set command aliases
func init() {
	for alias, cmd := range Aliases {
		c := Commands[cmd]
		c.Aliases = append(c.Aliases, alias)
	}
}
