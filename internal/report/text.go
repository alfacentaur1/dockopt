package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	blue   = "\x1b[34m"
	green  = "\x1b[32m"
)

type painter bool

func (p painter) paint(color, s string) string {
	if !p {
		return s
	}
	return color + s + reset
}

func (p painter) severity(s rules.Severity) string {
	label := fmt.Sprintf("%-7s", s)
	switch s {
	case rules.Error:
		return p.paint(red, label)
	case rules.Warning:
		return p.paint(yellow, label)
	default:
		return p.paint(blue, label)
	}
}

// Text writes a human readable report.
func Text(w io.Writer, files []File, opts Options) error {
	p := painter(opts.Color)
	for i, f := range files {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, p.paint(bold, f.Path))
		if f.Error != "" {
			fmt.Fprintf(w, "  %s %s\n", p.paint(red, "error"), f.Error)
			continue
		}
		for _, a := range f.Applied {
			fmt.Fprintf(w, "  %s %s %s\n", p.paint(green, "fixed"), p.paint(dim, a.RuleID), a.Message)
		}
		if len(f.Findings) == 0 && len(f.Applied) == 0 {
			fmt.Fprintf(w, "  %s\n", p.paint(green, "no issues"))
		}
		for _, fd := range f.Findings {
			fix := ""
			if fd.Fixable {
				fix = p.paint(dim, " (fixable)")
			}
			fmt.Fprintf(w, "  %4d  %s %s  %s%s\n", fd.Line, p.severity(fd.Severity), p.paint(dim, fd.RuleID), fd.Message, fix)
		}
		if f.Diff != "" {
			fmt.Fprintln(w)
			for _, line := range strings.Split(strings.TrimSuffix(f.Diff, "\n"), "\n") {
				switch {
				case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
					line = p.paint(green, line)
				case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
					line = p.paint(red, line)
				case strings.HasPrefix(line, "@@"):
					line = p.paint(blue, line)
				}
				fmt.Fprintln(w, "  "+line)
			}
		}
	}

	s := Summarize(files)
	fmt.Fprintln(w)
	fmt.Fprintln(w, summaryLine(s))
	return nil
}

func summaryLine(s Summary) string {
	parts := []string{plural(s.Files, "file")}
	if s.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", s.Failed))
	}
	parts = append(parts, plural(s.Errors, "error"), plural(s.Warnings, "warning"), plural(s.Infos, "info"))
	line := strings.Join(parts, ", ")
	if s.Fixed > 0 {
		line += fmt.Sprintf("; fixed %d", s.Fixed)
	}
	if s.Fixable > 0 {
		line += fmt.Sprintf("; %d fixable, run `dockopt fix -w`", s.Fixable)
	}
	return line
}

func plural(n int, word string) string {
	if n == 1 || word == "info" {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
