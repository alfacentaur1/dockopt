package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixPrintsTheFixedFile(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\nRUN apk add git\nUSER app\n"})
	code, out, _ := runCLI(t, "fix", dir)
	if code != exitOK || out != "FROM alpine:3.20\nRUN apk add --no-cache git\nUSER app\n" {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
	// stdout mode never touches the file
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if strings.Contains(string(b), "--no-cache") {
		t.Fatal("file was modified without -w")
	}
}

func TestFixWrite(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": goDockerfile, "go.mod": "module x\n", ".dockerignore": ".git\n"})
	code, out, _ := runCLI(t, "fix", "-w", dir)
	if code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	got := string(b)
	for _, want := range []string{"COPY go.mod ./\nRUN go mod download\nCOPY . .", `CMD ["/app"]`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(out, "fixed DO005") {
		t.Errorf("report should list the applied fixes:\n%s", out)
	}
}

func TestFixDiffAndCheck(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\nADD app /app\nUSER app\n"})

	code, out, _ := runCLI(t, "fix", "-diff", dir)
	if code != exitOK || !strings.Contains(out, "-ADD app /app") || !strings.Contains(out, "+COPY app /app") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
	if code, _, _ := runCLI(t, "fix", "-check", dir); code != exitFindings {
		t.Fatalf("-check should fail when something is fixable, got %d", code)
	}

	runCLI(t, "fix", "-w", dir)
	if code, _, _ := runCLI(t, "fix", "-check", dir); code != exitOK {
		t.Fatalf("-check should pass after fixing, got %d", code)
	}
}

func TestFixSeveralFilesNeedsAMode(t *testing.T) {
	a := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\n"})
	b := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\n"})
	if code, _, stderr := runCLI(t, "fix", a, b); code != exitError || !strings.Contains(stderr, "-w") {
		t.Fatalf("got %d %q", code, stderr)
	}
}
