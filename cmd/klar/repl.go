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
	fmt.Printf(`%sKlar %s%s
Type %[4]s'help'%[5]s for more information, or %[4]sCtrl+D%[5]s or %[4]s'exit'%[5]s to exit.
`, cli.ANSIBold+cli.ANSIYellow, version.KlarVersion,
		cli.ANSIReset+cli.ANSIGreen,
		cli.ANSICyan, cli.ANSIReset+cli.ANSIGreen,
	)
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(cli.ANSIGreen + "> " + cli.ANSIReset)
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
		RunString(input)
	}
}
