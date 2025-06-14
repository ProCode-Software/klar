package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
)

func showHelp() {
	fmt.Fprint(os.Stderr, HelpString)
}

func main() {
	os.Setenv("KLAR_REPL", "0")
	var strProg string
	flag.StringVar(&strProg, "c", "", "Program passed as string")
	flag.Usage = showHelp
	flag.Parse()
	if strProg != "" {
		RunString(strProg)
		os.Exit(0)
	}
	args := os.Args
	if len(args) <= 1 {
		tryPipe()
		showHelp()
		os.Exit(2)
	}
	cmd := args[1]
	switch cmd {
	case "run":
		if len(args) <= 2 {
			cli.Error("No file to run specified")
			os.Exit(2)
		}
		cmd = args[2]
		fallthrough
	default:
		RunFile(cmd)
	case "repl":
		StartRepl()
	case "test", "install", "build":
		cli.Error(fmt.Sprintf("Command '%s' is not implemented yet.", cmd), "")
	case "help", "--help", "-h":
		showHelp()
	}
}
