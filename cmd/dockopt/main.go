// Command dockopt analyzes and fixes Dockerfiles, measures the impact and
// comments on merge requests.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// version is set at build time: -ldflags "-X main.version=v1.0.0"
var version = "dev"

const usage = `dockopt finds and fixes problems in Dockerfiles.

Usage:
  dockopt <command> [flags] [paths...]

Commands:
  analyze   report problems (default command)
  fix       fix what can be fixed; prints the result, or -w to write it back
  audit     analyze every Dockerfile under a directory
  bench     build the original and the fixed image and compare them
  ci        analyze and comment on the current merge request
  rules     list all rules, or explain one: dockopt rules DO005
  version   print the version

Paths can be Dockerfiles or directories containing one. Run
"dockopt <command> -h" for the flags of a command.
`

// Exit codes.
const (
	exitOK       = 0
	exitFindings = 1 // findings at or above --fail-on
	exitError    = 2 // bad usage, unreadable or unparsable files, docker errors...
)

// errUsage is returned for bad command lines; the message is already printed.
var errUsage = errors.New("usage")

// app carries the output streams so tests can capture them.
type app struct {
	stdout, stderr io.Writer
	color          bool
}

func main() {
	a := &app{stdout: os.Stdout, stderr: os.Stderr, color: wantColor()}
	os.Exit(a.run(os.Args[1:]))
}

func (a *app) run(args []string) int {
	cmd := "analyze"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") && isCommand(args[0]) {
		cmd, args = args[0], args[1:]
	}

	var (
		code int
		err  error
	)
	switch cmd {
	case "analyze":
		code, err = a.analyze(args)
	case "fix":
		code, err = a.fix(args)
	case "audit":
		code, err = a.audit(args)
	case "bench":
		code, err = a.bench(args)
	case "ci":
		code, err = a.ci(args)
	case "rules":
		code, err = a.rules(args)
	case "version":
		fmt.Fprintln(a.stdout, "dockopt", version)
	case "help":
		fmt.Fprint(a.stdout, usage)
	}

	switch {
	case errors.Is(err, errUsage):
		return exitError
	case err != nil:
		fmt.Fprintln(a.stderr, "dockopt:", err)
		return exitError
	}
	return code
}

func isCommand(s string) bool {
	switch s {
	case "analyze", "fix", "audit", "bench", "ci", "rules", "version", "help":
		return true
	}
	return false
}

// wantColor enables colors only on a terminal, and never with NO_COLOR set.
func wantColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
