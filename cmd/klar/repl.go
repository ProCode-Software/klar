package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/version"
)

func StartRepl() {
	os.Setenv("KLAR_REPL", "1") // Prevent exiting on error
	fmt.Printf(
		`%sKlar %s%[5]s
Type %[4]s'help'%[5]s for more information. Press %[4]sCtrl+D%[5]s or %[4]s'exit'%[5]s to exit.
%[3]s`,
		cli.ANSIBold+cli.ANSIYellow, version.KlarVersion, cli.ANSIReset,
		cli.ANSIReset+cli.ANSICyan, cli.ANSIReset+cli.ANSIDim,
	)
	r := bufio.NewReader(os.Stdin)
	File = "<repl>"
	for {
		fmt.Print(cli.ANSIMagenta + "> " + cli.ANSIReset)
		input, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			cli.InternalError(err)
		}
		input = strings.TrimSpace(input)
		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}
		RunString(input)
	}
}
