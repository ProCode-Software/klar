package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
)

func showHelp() {
	fmt.Fprint(os.Stderr, HelpString)
	os.Exit(2)
}

func main() {
	var strProg string
	flag.StringVar(&strProg, "c", "", "Program passed as string")
	flag.Parse()
	if strProg != "" {
		RunString(strProg)
		os.Exit(0)
	}
	args := os.Args
	if len(args) <= 1 {
		tryPipe()
		showHelp()
	}
	cmd := args[1]
	switch cmd {
	case "run":
		if len(args) <= 2 {
			fmt.Fprintln(os.Stderr, "Error: No file specified")
			os.Exit(2)
		}
		cmd = args[2]
		fallthrough
	default:
		RunFile(cmd)
	case "repl", "test", "install", "build":
		cli.Fail(fmt.Sprintf("Command '%s' is not implemented yet.", cmd), "")
	case "help", "--help", "-h":
		showHelp()
	}
}
