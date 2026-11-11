package core

import (
	"sort"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

// Options tune which findings come out.
type Options struct {
	// Ignore lists rule IDs to skip entirely.
	Ignore []string
	// MinSeverity drops findings below it.
	MinSeverity rules.Severity
}

func (o Options) ignored(id string) bool {
	for _, i := range o.Ignore {
		if strings.EqualFold(strings.TrimSpace(i), id) {
			return true
		}
	}
	return false
}

// Analysis is the result of Analyze.
type Analysis struct {
	Findings []rules.Finding `json:"findings"`
	Lang     string          `json:"language,omitempty"`
}

// Analyze runs every rule over the Dockerfile.
func Analyze(in Input, opts Options) (*Analysis, error) {
	ctx, err := newContext(in, in.Dockerfile)
	if err != nil {
		return nil, err
	}
	return &Analysis{Findings: run(ctx, opts), Lang: ctx.Lang}, nil
}

// run checks all rules, applies options and inline suppressions, and sorts
// the findings by line.
func run(ctx *rules.Context, opts Options) []rules.Finding {
	sup := findSuppressions(ctx.File)
	var out []rules.Finding
	for _, r := range rules.All() {
		if opts.ignored(r.ID()) {
			continue
		}
		for _, f := range r.Check(ctx) {
			if f.Severity >= opts.MinSeverity && sup.allows(f) {
				out = append(out, f)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].RuleID < out[j].RuleID
	})
	return out
}
