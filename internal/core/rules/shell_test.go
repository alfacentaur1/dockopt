package rules

import (
	"reflect"
	"regexp"
	"testing"
)

func TestCommands(t *testing.T) {
	got := commands("apt-get update &&   apt-get install -y git \\\n || true; echo done | tee log")
	want := []string{"apt-get update", "apt-get install -y git", "true", "echo done", "tee log"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestInsertAfter(t *testing.T) {
	re := regexp.MustCompile(`apt-get install`)
	got, ok := insertAfter("RUN apt-get install -y git", re, "--no-install-recommends")
	if !ok || got != "RUN apt-get install --no-install-recommends -y git" {
		t.Fatalf("got %q, %v", got, ok)
	}
	if _, ok := insertAfter("RUN true", re, "x"); ok {
		t.Fatal("expected no match")
	}
}

func TestAppendCommand(t *testing.T) {
	if got := appendCommand("RUN a", "b"); got != "RUN a && b" {
		t.Errorf("single line: got %q", got)
	}
	got := appendCommand("RUN a \\\n  && b  ", "c")
	if want := "RUN a \\\n  && b \\\n  && c"; got != want {
		t.Errorf("multi line: got %q, want %q", got, want)
	}
}

func TestReplaceKeyword(t *testing.T) {
	if got := replaceKeyword("add --chown=app src /dst", "COPY"); got != "COPY --chown=app src /dst" {
		t.Fatalf("got %q", got)
	}
}
