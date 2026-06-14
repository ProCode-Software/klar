package klarcmd

var KlarCommandAliases = map[string]string{
	"b": "build",
	"r": "run",
}

// Set command aliases
func init() {
	for alias, cmd := range KlarCommandAliases {
		c := KlarCommands[cmd]
		c.Aliases = append(c.Aliases, alias)
	}
}
