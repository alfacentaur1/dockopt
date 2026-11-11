package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI runs dockopt with args and returns the exit code and output.
func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	a := &app{stdout: &out, stderr: &errOut}
	code := a.run(args)
	return code, out.String(), errOut.String()
}

// project writes files into a temp dir and returns its path.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const goDockerfile = `FROM golang:1.22
WORKDIR /src
COPY . .
RUN go build -o /app .
CMD /app
`

func TestAnalyzeReportsAndFails(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": goDockerfile, "go.mod": "module x\n"})

	code, out, _ := runCLI(t, "analyze", dir)
	if code != exitFindings {
		t.Fatalf("exit code = %d, want %d\n%s", code, exitFindings, out)
	}
	for _, id := range []string{"DO005", "DO006", "DO008", "DO010", "DO014"} {
		if !strings.Contains(out, id) {
			t.Errorf("expected %s in output:\n%s", id, out)
		}
	}
}

func TestAnalyzeIsTheDefaultCommand(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\nUSER app\nCMD [\"sh\"]\n"})
	code, out, _ := runCLI(t, filepath.Join(dir, "Dockerfile"))
	if code != exitOK || !strings.Contains(out, "no issues") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
}

func TestAnalyzeFailOnAndIgnore(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine\nUSER app\n"})

	if code, _, _ := runCLI(t, "analyze", "--fail-on", "error", dir); code != exitOK {
		t.Errorf("only warnings, --fail-on error should pass, got %d", code)
	}
	if code, _, _ := runCLI(t, "analyze", "--ignore", "DO001", dir); code != exitOK {
		t.Errorf("the only finding is ignored, got %d", code)
	}
	if code, _, stderr := runCLI(t, "analyze", "--ignore", "DO999", dir); code != exitError || !strings.Contains(stderr, "unknown rule") {
		t.Errorf("unknown rule should be an error, got %d %q", code, stderr)
	}
}

func TestAnalyzeMissingFile(t *testing.T) {
	code, out, _ := runCLI(t, "analyze", filepath.Join(t.TempDir(), "nope"))
	if code != exitError || !strings.Contains(out, "error") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
}

func TestAnalyzeJSON(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine\nUSER app\n"})
	_, out, _ := runCLI(t, "analyze", "--format", "json", dir)
	if !strings.Contains(out, `"rule": "DO001"`) || !strings.Contains(out, `"summary"`) {
		t.Fatalf("unexpected json:\n%s", out)
	}
}

func TestRulesCommand(t *testing.T) {
	code, out, _ := runCLI(t, "rules")
	if code != exitOK || !strings.Contains(out, "DO001") || !strings.Contains(out, "DO015") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
	_, out, _ = runCLI(t, "rules", "do005")
	if !strings.Contains(out, "fixable:  yes") {
		t.Fatalf("explain output:\n%s", out)
	}
}

func TestBadFlag(t *testing.T) {
	if code, _, _ := runCLI(t, "analyze", "--nope"); code != exitError {
		t.Fatalf("got %d", code)
	}
}
