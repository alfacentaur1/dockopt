package main

import (
	"strings"
	"testing"
)

func TestIsDockerfile(t *testing.T) {
	for name, want := range map[string]bool{
		"Dockerfile":              true,
		"Dockerfile.prod":         true,
		"api.Dockerfile":          true,
		"build.dockerfile":        true,
		"Dockerfile.dockerignore": false,
		"docker-compose.yml":      false,
	} {
		if got := isDockerfile(name); got != want {
			t.Errorf("isDockerfile(%q) = %v", name, got)
		}
	}
}

func TestAudit(t *testing.T) {
	dir := project(t, map[string]string{
		"api/Dockerfile":                  "FROM golang:1.22\nRUN go build\n",
		"web/Dockerfile.prod":             "FROM node:20\nUSER node\nCMD [\"node\", \"x.js\"]\n",
		"node_modules/pkg/Dockerfile":     "FROM nope\n",
		"worker/worker.Dockerfile":        "FROM alpine:3.20\nUSER app\n",
		"docs/Dockerfile.md.dockerignore": "x\n",
	})

	code, out, _ := runCLI(t, "audit", dir)
	if code != exitFindings {
		t.Fatalf("code %d\n%s", code, out)
	}
	if strings.Contains(out, "node_modules") {
		t.Error("node_modules must be skipped")
	}
	for _, want := range []string{"api/Dockerfile", "web/Dockerfile.prod", "worker/worker.Dockerfile", "3 files", "Most common:", "DO008"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestAuditEmpty(t *testing.T) {
	if code, _, stderr := runCLI(t, "audit", t.TempDir()); code != exitError || !strings.Contains(stderr, "no Dockerfiles") {
		t.Fatalf("got %d %q", code, stderr)
	}
}
