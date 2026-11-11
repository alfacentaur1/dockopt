package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/report"
)

// common holds the flags most commands share.
type common struct {
	format   string
	ignore   string
	severity string
	failOn   string
	noColor  bool
	context  string
}

func (c *common) register(fs *flag.FlagSet, withFailOn bool) {
	fs.StringVar(&c.format, "format", "text", "output format: text, json or md")
	fs.StringVar(&c.ignore, "ignore", "", "comma separated rule IDs to skip, e.g. DO001,DO004")
	fs.StringVar(&c.severity, "severity", "info", "hide findings below this severity: info, warning, error")
	fs.BoolVar(&c.noColor, "no-color", false, "disable colors in text output")
	fs.StringVar(&c.context, "context", "", "build context directory (default: the Dockerfile's directory)")
	if withFailOn {
		fs.StringVar(&c.failOn, "fail-on", "warning", "exit with 1 when a finding has at least this severity, or \"never\"")
	}
}

func (c *common) options() (core.Options, error) {
	sev, err := rules.ParseSeverity(c.severity)
	if err != nil {
		return core.Options{}, err
	}
	var ignore []string
	for _, id := range strings.Split(c.ignore, ",") {
		if id = strings.TrimSpace(id); id != "" {
			if rules.Get(id) == nil {
				return core.Options{}, fmt.Errorf("unknown rule %q in --ignore", id)
			}
			ignore = append(ignore, strings.ToUpper(id))
		}
	}
	return core.Options{Ignore: ignore, MinSeverity: sev}, nil
}

// exitCode decides between exitOK and exitFindings.
func (c *common) exitCode(files []report.File) (int, error) {
	for _, f := range files {
		if f.Error != "" {
			return exitError, nil
		}
	}
	if c.failOn == "never" || c.failOn == "" {
		return exitOK, nil
	}
	threshold, err := rules.ParseSeverity(c.failOn)
	if err != nil {
		return exitError, err
	}
	for _, f := range files {
		for _, fd := range f.Findings {
			if fd.Severity >= threshold {
				return exitFindings, nil
			}
		}
	}
	return exitOK, nil
}

// newFlags creates a FlagSet that prints usage to stderr and never exits.
func newFlags(a *app, name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	fs.Usage = func() {
		fmt.Fprintf(a.stderr, "Usage: dockopt %s %s\n\nFlags:\n", name, synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// parse wraps fs.Parse and turns -h and bad flags into errUsage.
func parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return errUsage
	}
	return nil
}

func (a *app) write(w io.Writer, format string, files []report.File, noColor bool) error {
	return report.Write(w, format, files, report.Options{Color: a.color && !noColor})
}
