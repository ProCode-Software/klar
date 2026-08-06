package run

import "github.com/ProCode-Software/klar/pkg/argparse"

var Flags = argparse.NewParser("[input]").
	StringFlag("runtime", "The JavaScript runtime to use", "command", "", "R")
