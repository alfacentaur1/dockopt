package rules

import (
	"strings"
	"testing"

	"github.com/alfacentaur1/dockopt/internal/core/render"
	"github.com/alfacentaur1/dockopt/internal/parser"
)

func TestParseSeverity(t *testing.T) {
	for in, want := range map[string]Severity{"info": Info, "WARN": Warning, "warning": Warning, " error ": Error} {
		got, err := ParseSeverity(in)
		if err != nil || got != want {
			t.Errorf("ParseSeverity(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseSeverity("fatal"); err == nil {
		t.Error("expected error for unknown severity")
	}
}

func TestAllIsSortedAndIDsLookStable(t *testing.T) {
	all := All()
	for i, r := range all {
		if !strings.HasPrefix(r.ID(), "DO") || len(r.ID()) != 5 {
			t.Errorf("rule id %q doesn't look like DOnnn", r.ID())
		}
		if r.Title() == "" || r.Description() == "" {
			t.Errorf("%s needs a title and a description", r.ID())
		}
		if i > 0 && all[i-1].ID() >= r.ID() {
			t.Errorf("rules not sorted: %s before %s", all[i-1].ID(), r.ID())
		}
	}
}

// helpers shared by the rule tests

func ctxFor(t *testing.T, src string) *Context {
	t.Helper()
	df, err := parser.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return &Context{File: df}
}

func check(t *testing.T, id string, ctx *Context) []Finding {
	t.Helper()
	r := Get(id)
	if r == nil {
		t.Fatalf("rule %s is not registered", id)
	}
	return r.Check(ctx)
}

// fixOnce applies the rule's fix to its first finding and returns the
// rendered result. It fails the test if there was nothing to fix.
func fixOnce(t *testing.T, id string, ctx *Context) string {
	t.Helper()
	findings := check(t, id, ctx)
	if len(findings) == 0 {
		t.Fatalf("%s: no findings to fix", id)
	}
	fx, ok := Get(id).(Fixer)
	if !ok {
		t.Fatalf("%s is not fixable", id)
	}
	if !fx.Fix(ctx, findings[0]) {
		t.Fatalf("%s: fix returned false", id)
	}
	return string(render.Render(ctx.File))
}

// assertFixed applies one fix and compares the whole file.
func assertFixed(t *testing.T, id, src, want string) {
	t.Helper()
	if got := fixOnce(t, id, ctxFor(t, src)); got != want {
		t.Fatalf("%s fix:\n--- got\n%s--- want\n%s", id, got, want)
	}
}

// assertFindings checks how many findings a rule reports.
func assertFindings(t *testing.T, id string, ctx *Context, n int) []Finding {
	t.Helper()
	got := check(t, id, ctx)
	if len(got) != n {
		t.Fatalf("%s: got %d findings, want %d: %+v", id, len(got), n, got)
	}
	return got
}
