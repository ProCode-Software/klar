package main

var Aliases = map[string]string{
	"b":     "build",
	"r":     "run",
	"up":    "upgrade",
	"check": "lint",
}

// Set command aliases
func init() {
	for alias, cmd := range Aliases {
		c := Commands[cmd]
		c.Aliases = append(c.Aliases, alias)
	}
}
