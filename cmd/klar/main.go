package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/commands"
)

func showHelp() {
	fmt.Fprint(os.Stderr, HelpString)
	os.Exit(2)
}

func main() {
	flag.Parse()
	args := os.Args
	if len(args) <= 1 {
		showHelp()
	}
	cmd := args[1]
	switch cmd {
	case "help", "--help", "-h":
		showHelp()
	case "build":
		commands.Build()
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\nRun 'klar help' for usage.\n", cmd)
		os.Exit(2)
	}
}
