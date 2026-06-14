package klarcmd

import (
	"github.com/ProCode-Software/klar/cmd/klar/internal/klarcmd"
	"github.com/ProCode-Software/klar/internal/command"
)

func LookupKlarCmd(cmd string) *command.Command {
	return command.Lookup(cmd, klarcmd.KlarCommands, klarcmd.KlarCommandAliases)
}
