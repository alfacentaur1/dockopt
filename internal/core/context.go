// Package core is the pure part of dockopt: bytes in, findings and bytes out.
// No I/O, no docker, no filesystem, so the same code runs in the CLI, in CI
// and compiled to WebAssembly in the browser.
package core

import (
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/model"
	"github.com/alfacentaur1/dockopt/internal/parser"
)

// Input is everything the core needs to know about one Dockerfile.
type Input struct {
	Dockerfile []byte
	// DockerIgnore is the content of .dockerignore, nil when there is none.
	DockerIgnore []byte
	// Files are names in the build context, relative and slash separated.
	// Only used for language detection and to tune fixes; may be empty.
	Files []string
}

// newContext parses src and builds the rule context around it.
func newContext(in Input, src []byte) (*rules.Context, error) {
	df, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	return &rules.Context{
		File:            df,
		DockerIgnore:    parseIgnore(in.DockerIgnore),
		HasDockerIgnore: in.DockerIgnore != nil,
		Files:           in.Files,
		Lang:            DetectLanguage(in.Files),
	}, nil
}

// parseIgnore returns .dockerignore patterns without comments and blanks.
func parseIgnore(b []byte) []string {
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

// markers are files whose presence at the root of the context gives the
// project language away.
var markers = []struct{ file, lang string }{
	{"go.mod", "go"},
	{"package.json", "node"},
	{"requirements.txt", "python"},
	{"pyproject.toml", "python"},
	{"Pipfile", "python"},
	{"pom.xml", "java"},
	{"build.gradle", "java"},
	{"build.gradle.kts", "java"},
}

// DetectLanguage guesses the project language from the build context.
// It returns "" when it can't tell; rules then fall back to the base image.
func DetectLanguage(files []string) string {
	have := make(map[string]bool, len(files))
	for _, f := range files {
		have[f] = true
	}
	for _, m := range markers {
		if have[m.file] {
			return m.lang
		}
	}
	return ""
}

// Parse is exported for callers that need the model, e.g. the bench.
func Parse(src []byte) (*model.Dockerfile, error) { return parser.Parse(src) }
