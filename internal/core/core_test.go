package core

import (
	"testing"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

const sample = `FROM node:latest
WORKDIR /app
COPY . .
RUN npm ci
RUN npm run build
CMD npm start
`

func ids(fs []rules.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.RuleID)
	}
	return out
}

func has(fs []rules.Finding, id string) bool {
	for _, f := range fs {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

func TestAnalyzeFindsAndSortsByLine(t *testing.T) {
	a, err := Analyze(Input{Dockerfile: []byte(sample), Files: []string{"package.json"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Lang != "node" {
		t.Errorf("lang = %q", a.Lang)
	}
	for _, id := range []string{"DO001", "DO004", "DO005", "DO006", "DO008", "DO014"} {
		if !has(a.Findings, id) {
			t.Errorf("expected %s in %v", id, ids(a.Findings))
		}
	}
	for i := 1; i < len(a.Findings); i++ {
		if a.Findings[i-1].Line > a.Findings[i].Line {
			t.Fatalf("not sorted by line: %v", a.Findings)
		}
	}
}

func TestAnalyzeOptions(t *testing.T) {
	a, _ := Analyze(Input{Dockerfile: []byte(sample)}, Options{Ignore: []string{"do001"}, MinSeverity: rules.Warning})
	if has(a.Findings, "DO001") {
		t.Error("DO001 should be ignored")
	}
	if has(a.Findings, "DO004") {
		t.Error("DO004 is info and should be filtered out")
	}
}

func TestInlineSuppressions(t *testing.T) {
	src := "# dockopt:ignore-file DO008\n# dockopt:ignore DO001\nFROM node:latest\n# dockopt:ignore\nCMD npm start\n"
	a, err := Analyze(Input{Dockerfile: []byte(src)}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Findings) != 0 {
		t.Fatalf("everything is suppressed, got %v", ids(a.Findings))
	}
}
