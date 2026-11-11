// Package report formats results as plain text, JSON or Markdown.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

// File is the result for one Dockerfile.
type File struct {
	Path     string          `json:"path"`
	Lang     string          `json:"language,omitempty"`
	Findings []rules.Finding `json:"findings"`
	Applied  []core.Applied  `json:"applied,omitempty"`
	Diff     string          `json:"diff,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// Summary counts findings across files.
type Summary struct {
	Files    int `json:"files"`
	Failed   int `json:"failed"` // files that couldn't be read or parsed
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Infos    int `json:"infos"`
	Fixable  int `json:"fixable"`
	Fixed    int `json:"fixed"`
}

// Total is the number of findings.
func (s Summary) Total() int { return s.Errors + s.Warnings + s.Infos }

// Summarize counts findings.
func Summarize(files []File) Summary {
	s := Summary{Files: len(files)}
	for _, f := range files {
		if f.Error != "" {
			s.Failed++
		}
		s.Fixed += len(f.Applied)
		for _, fd := range f.Findings {
			switch fd.Severity {
			case rules.Error:
				s.Errors++
			case rules.Warning:
				s.Warnings++
			default:
				s.Infos++
			}
			if fd.Fixable {
				s.Fixable++
			}
		}
	}
	return s
}

// Formats lists the accepted output formats.
var Formats = []string{"text", "json", "md"}

// Options change how text output looks.
type Options struct {
	Color bool
}

// Write prints the results in the given format.
func Write(w io.Writer, format string, files []File, opts Options) error {
	switch format {
	case "", "text":
		return Text(w, files, opts)
	case "json":
		return JSON(w, files)
	case "md", "markdown":
		return Markdown(w, files)
	}
	return fmt.Errorf("unknown format %q (use %s)", format, strings.Join(Formats, ", "))
}

// JSON writes {"files": [...], "summary": {...}}.
func JSON(w io.Writer, files []File) error {
	if files == nil {
		files = []File{}
	}
	for i := range files {
		if files[i].Findings == nil {
			files[i].Findings = []rules.Finding{}
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Files   []File  `json:"files"`
		Summary Summary `json:"summary"`
	}{files, Summarize(files)})
}
