package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"slices"
	"time"

	"github.com/ProCode-Software/klar/cmd/glas"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/run"
	"github.com/ProCode-Software/klar/internal/util"
)

func main() {
	// Write panics to a file. If there's no panic, delete it before exiting.
	crashLogFile := setCrashOutput()
	// Also, the CLI uses [cli.SignalExit] to exit instead of [os.Exit] so deferred
	// functions can run before exiting.
	defer handleExit(crashLogFile)

	if profFile := startProf(); profFile != nil {
		defer profFile.Close()
		defer pprof.StopCPUProfile()
	}
	args := os.Args
	if len(args) < 2 {
		tryPipe()
		ShowHelp(os.Stderr, false)
		cli.Exit(2)
	}
	cmdName := args[1]
	switch cmdName {
	case "-":
		tryPipe()
		command.Run(Commands["repl"])
	case "-c":
		if len(args) < 3 {
			cli.Failure(
				"Expected program as string\n\nUsage: ",
				ansi.BoldMagenta("klar ")+ansi.Cyan("-c ")+ansi.Blue("<program>\n\n"),
				"Use "+ansi.Cyan("'klar --help'")+" for more information.",
			)
			cli.Exit(2)
		}
		if _, err := run.RunString(args[2], "string"); err != nil {
			cli.FailureError(err)
		}
	case "--help", "-h":
		ShowHelp(os.Stdout, true)
	case "-v", "--version", "version":
		fmt.Printf("Klar %s (%s)\n", cli.KlarVersion, cli.KlarCommit)
	case "test", "lint", "generate":
		// Unimplemented command
		cli.Failure(ansi.ColorSprintf(
			ansi.CodeBold,
			"Command %s isn't implemented yet", ansi.Cyan(cmdName),
		))
	case "glas":
		os.Args = slices.Delete(os.Args, 1, 2) // Strip 'klar' from 'klar glas'
		glas.Main(func(cmdName string) *command.Command {
			return command.Lookup(cmdName, Commands, Aliases)
		})
	case "help", "h", "?":
		// klar help | klar help klar
		if len(args) < 3 || args[2] == "" || args[2] == "klar" {
			ShowHelp(os.Stdout, true)
			cli.Exit(0)
		}
		// klar help cmd -> klar cmd --help
		cmd := args[2]
		if command.Lookup(cmd, Commands, Aliases) != nil {
			os.Args[1], cmdName = cmd, cmd
			os.Args[2] = "--help"
		}
		fallthrough
	default:
		// `klar -` is already handled above
		if badFlag := args[1]; badFlag[0] == '-' {
			// Invalid flag
			cli.ColorErrorfln("<**>I don't understand the <c>%s</c> flag</**>", badFlag)
			FlagHelp(command.NewHelpBuilder(os.Stderr))
			cli.Exit(2)
		}
		// Command
		if cmd := command.Lookup(cmdName, Commands, Aliases); cmd != nil {
			command.Run(cmd)
			break
		}
		// Equivalent to `klar run [file]`
		os.Args = append([]string{"klar", ""}, os.Args[1:]...)
		command.Run(Commands["run"])
	}
}

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Pipe
	run.RunInput(os.Stdin, "standardInput")
	cli.Exit(0)
}

// Debugging/Panics
// =======

func handleExit(crashLogFile *os.File) {
	if crashLogFile != nil {
		if err := crashLogFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write crash log file: %v", err)
		}
	}
	switch r := recover().(type) {
	case cli.SignalExit:
		if crashLogFile != nil {
			os.Remove(crashLogFile.Name())
		}
		os.Exit(r.Code)
	case nil:
		if crashLogFile != nil {
			os.Remove(crashLogFile.Name())
		}
		return
	default:
		showPanicMessage(crashLogFile)
		panic(r)
	}
}

const panicMessage = `<** r!>Oh no!</r!> The Klar CLI has crashed.</>
This isn't your fault; plase report an issue at <c!>%s/new</c!>.

<**>In your report, please include:</**>

  - The crash message below
  - The code you were running
  - Steps to reproduce this <d>(what you did to get here)</d>
  - Klar version: <b>v%s</b>
  - Operating system: <b>%s/%s</b>
  
We recommend re-running the command with the <c!>KLAR_TRACE=1</c!> environment variable set to show more details that will help us diagnose the issue.

%s

<r!>The crash message was:</r!>`

func showPanicMessage(crashLogFile *os.File) {
	var crashLogPathMsg string
	if crashLogFile != nil {
		crashLogPathMsg = "This crash output has been saved to " +
			ansi.BrightCyan(util.ShortenHomePath(crashLogFile.Name())) + "."
	}
	ansi.TagFprintfln(
		os.Stderr, panicMessage, cli.KlarIssues,
		cli.KlarVersionAndCommit, runtime.GOOS, runtime.GOARCH,
		crashLogPathMsg,
	)
}

// If a profile was started, startProf returns the file the pprof
// is writing to.
func startProf() *os.File {
	v := os.Getenv("KLAR_PROFILE")
	var path string
	switch v {
	case "", "0":
		return nil
	case "1":
		path = "klar.prof"
	default:
		path = v
	}
	file, err := os.Create(path) //nolint:gosec
	if err != nil {
		panic(err)
	}
	runtime.SetCPUProfileRate(10_000)
	if err = pprof.StartCPUProfile(file); err != nil {
		panic(err)
	}
	return file
}

// setCrashOutput sets the crash output to a file in the Klar state directory.
// The time used in the filename is based on the time the executable was started,
// rather than the time the crash occurred.
//
// setCrashOutput also deletes crash log files older than the latest 4.
//
// TODO: We could probably rename the file
func setCrashOutput() (f *os.File) {
	stateDir, err := module.KlarStateDir()
	if err != nil {
		// Don't crash the program if we can't create the file
		fmt.Fprintf(
			os.Stderr, "Failed to get Klar state directory for crash output: %v", err,
		)
		return nil
	}
	crashLogDir := filepath.Join(stateDir, "crashDumps")
	stat, err := os.Stat(crashLogDir)
	switch {
	case err == nil && !stat.IsDir():
		// Delete it and recreate it as a directory
		if err := os.Remove(crashLogDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete file %q: %v", crashLogDir, err)
		}
		fallthrough
	case err != nil && errors.Is(err, fs.ErrNotExist): // Doesn't exist
		if err := os.MkdirAll(crashLogDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %q: %v", crashLogDir, err)
			return nil
		}
	case err != nil:
		// Other error. We'll still continue. An error may be reported when trying
		// to create a new log file.
		fmt.Fprintf(
			os.Stderr, "Failed to stat Klar crash log directory %q: %v",
			crashLogDir, err,
		)
	case stat.IsDir():
		deleteOldCrashLogs(crashLogDir)
	}

	// Create a new crash log file. It will be deleted if there is no panic.
	f, err = os.Create(filepath.Join(
		stateDir, "crashDumps",
		fmt.Sprintf("klarPanic_%s.txt", time.Now().Format("2006-01-02_15-04-05")),
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Klar crash output: %v", err)
		return nil
	}
	debug.SetCrashOutput(f, debug.CrashOptions{})
	return f
}

func deleteOldCrashLogs(crashLogDir string) {
	// Crash logs older than the latest maxCrashLogs will be deleted
	const maxCrashLogs = 4

	items, err := os.ReadDir(crashLogDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read crash log directory: %v", err)
		return // We can create a new one, but old crash logs won't be deleted
	}
	// Use '<' instead of '<=' because a new log will be created immediately after this
	if len(items) < maxCrashLogs {
		return
	}
	// Newest times are first
	slices.SortFunc(items, func(a, b os.DirEntry) int {
		infoA, errA := a.Info()
		infoB, errB := b.Info()
		if errA != nil || errB != nil {
			return 0
		}
		return infoB.ModTime().Compare(infoA.ModTime())
	})
	// Delete the oldest crash logs
	for _, item := range items[maxCrashLogs-1:] { // Always >= 4
		if err := os.Remove(filepath.Join(crashLogDir, item.Name())); err != nil {
			fmt.Fprintf(
				os.Stderr, "Failed to delete crash log at %q: %v",
				filepath.Join(crashLogDir, item.Name()), err,
			)
		}
	}
}
