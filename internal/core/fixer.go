package core

import (
	"bytes"
	"fmt"

	"github.com/alfacentaur1/dockopt/internal/core/render"
	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/parser"
)

// MaxPasses bounds the fixer. A pass applies every fix it can, one at a
// time; the fixer stops after a pass that changes nothing, or after this many.
const MaxPasses = 10

// Applied describes one fix that was made.
type Applied struct {
	RuleID  string `json:"rule"`
	Line    int    `json:"line"` // line in the file as it was when the fix ran
	Message string `json:"message"`
}

// FixResult is the result of Fix.
type FixResult struct {
	Output    []byte          `json:"-"`
	Applied   []Applied       `json:"applied"`
	Remaining []rules.Finding `json:"remaining"`
	Lang      string          `json:"language,omitempty"`
}

// Changed reports whether any fix was applied.
func (r *FixResult) Changed() bool { return len(r.Applied) > 0 }

// Fix applies fixes one at a time and re-parses after each one, because two
// fixes touching the same instruction would otherwise fight.
func Fix(in Input, opts Options) (*FixResult, error) {
	cur := in.Dockerfile
	if _, err := parser.Parse(cur); err != nil {
		return nil, err
	}

	var applied []Applied
	for pass := 0; pass < MaxPasses; pass++ {
		progressed := false
		tried := map[string]bool{} // findings already attempted in this pass

		for {
			ctx, err := newContext(in, cur)
			if err != nil {
				return nil, err
			}
			f, ok := nextFixable(run(ctx, opts), tried)
			if !ok {
				break
			}
			tried[key(f)] = true

			fx, _ := rules.Get(f.RuleID).(rules.Fixer)
			if fx == nil || !fx.Fix(ctx, f) {
				continue
			}
			out := render.Render(ctx.File)
			if bytes.Equal(out, cur) {
				continue
			}
			if _, err := parser.Parse(out); err != nil {
				continue // never hand back something buildkit can't read
			}
			cur = out
			progressed = true
			applied = append(applied, Applied{RuleID: f.RuleID, Line: f.Line, Message: f.Message})
		}
		if !progressed {
			break
		}
	}

	ctx, err := newContext(in, cur)
	if err != nil {
		return nil, err
	}
	return &FixResult{Output: cur, Applied: applied, Remaining: run(ctx, opts), Lang: ctx.Lang}, nil
}

func key(f rules.Finding) string { return fmt.Sprintf("%s:%d", f.RuleID, f.Line) }

func nextFixable(fs []rules.Finding, tried map[string]bool) (rules.Finding, bool) {
	for _, f := range fs {
		if f.Fixable && !tried[key(f)] {
			return f, true
		}
	}
	return rules.Finding{}, false
}
