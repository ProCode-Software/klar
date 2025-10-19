package klarcmd

var KlarCommandAliases = map[string]string{}

// Set command aliases
func init() {
	for alias, cmd := range KlarCommandAliases {
		c := KlarCommands[cmd]
		KlarCommands[alias] = c
		c.Aliases = append(c.Aliases, alias)
	}
}
