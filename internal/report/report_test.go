package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

var sample = []File{
	{
		Path: "Dockerfile",
		Findings: []rules.Finding{
			{RuleID: "DO001", Severity: rules.Warning, Line: 1, Message: "alpine has no tag"},
			{RuleID: "DO013", Severity: rules.Warning, Line: 2, Message: "apk add without --no-cache", Fixable: true},
			{RuleID: "DO004", Severity: rules.Info, Line: 3, Message: "merge | these"},
		},
	},
	{Path: "api/Dockerfile", Applied: []core.Applied{{RuleID: "DO007", Line: 4, Message: "ADD of local files"}}},
	{Path: "broken/Dockerfile", Error: "dockerfile parse error"},
}

func TestSummarize(t *testing.T) {
	s := Summarize(sample)
	if s.Files != 3 || s.Failed != 1 || s.Warnings != 2 || s.Infos != 1 || s.Fixable != 1 || s.Fixed != 1 {
		t.Fatalf("unexpected summary %+v", s)
	}
}

func TestText(t *testing.T) {
	var b bytes.Buffer
	if err := Text(&b, sample, Options{}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"Dockerfile\n",
		"     2  warning DO013  apk add without --no-cache (fixable)",
		"fixed DO007 ADD of local files",
		"error dockerfile parse error",
		"3 files, 1 failed, 0 errors, 2 warnings, 1 info; fixed 1; 1 fixable",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("no colors were requested")
	}
}

func TestJSON(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, "json", sample, Options{}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Files []struct {
			Path     string `json:"path"`
			Findings []struct {
				Rule     string `json:"rule"`
				Severity string `json:"severity"`
			} `json:"findings"`
		} `json:"files"`
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, b.String())
	}
	if got.Files[0].Findings[0].Severity != "warning" || got.Summary.Warnings != 2 {
		t.Fatalf("unexpected json: %s", b.String())
	}
	if got.Files[1].Findings == nil {
		t.Error("findings should be [] rather than null")
	}
}

func TestMarkdown(t *testing.T) {
	var b bytes.Buffer
	files := append([]File{}, sample...)
	files[0].Diff = "--- a\n+++ b\n"
	if err := Write(&b, "md", files, Options{}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"### dockopt", "| 2 | 🟡 warning | `DO013` |", `merge \| these`, "```diff\n--- a"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestUnknownFormat(t *testing.T) {
	if err := Write(&bytes.Buffer{}, "xml", nil, Options{}); err == nil {
		t.Fatal("expected an error")
	}
}
