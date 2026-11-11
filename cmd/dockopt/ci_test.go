package main

import (
	"strings"
	"testing"
)

func TestCIPrintsTheCommentOutsideCI(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\nRUN apk add git\n"})

	code, out, _ := runCLI(t, "ci", dir)
	if code != exitFindings {
		t.Fatalf("code %d\n%s", code, out)
	}
	for _, want := range []string{"### dockopt", "`DO013`", "```diff", "+RUN apk add --no-cache git"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCIWithoutSuggestions(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\nRUN apk add git\nUSER app\n"})
	_, out, _ := runCLI(t, "ci", "-provider", "none", "-suggest=false", "-fail-on", "never", dir)
	if strings.Contains(out, "```diff") {
		t.Errorf("no diff was requested:\n%s", out)
	}
}

func TestCIUnknownProvider(t *testing.T) {
	dir := project(t, map[string]string{"Dockerfile": "FROM alpine:3.20\n"})
	if code, _, stderr := runCLI(t, "ci", "-provider", "bitbucket", dir); code != exitError || !strings.Contains(stderr, "unknown provider") {
		t.Fatalf("got %d %q", code, stderr)
	}
}
