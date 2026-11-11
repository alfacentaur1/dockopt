package core

import (
	"reflect"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := map[string][]string{
		"go":     {"main.go", "go.mod", "go.sum"},
		"node":   {"package.json", "src/index.js"},
		"python": {"pyproject.toml"},
		"java":   {"pom.xml"},
		"":       {"Makefile", "src/go.mod"}, // only the root counts
	}
	for want, files := range cases {
		if got := DetectLanguage(files); got != want {
			t.Errorf("DetectLanguage(%v) = %q, want %q", files, got, want)
		}
	}
}

func TestParseIgnore(t *testing.T) {
	got := parseIgnore([]byte("# comment\n.git\n\n  node_modules  \n"))
	if want := []string{".git", "node_modules"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNewContextKnowsAboutDockerignore(t *testing.T) {
	ctx, err := newContext(Input{DockerIgnore: []byte{}}, []byte("FROM alpine:3.20\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !ctx.HasDockerIgnore {
		t.Error("an empty .dockerignore still exists")
	}

	ctx, _ = newContext(Input{}, []byte("FROM alpine:3.20\n"))
	if ctx.HasDockerIgnore {
		t.Error("nil means there is no .dockerignore")
	}
}
