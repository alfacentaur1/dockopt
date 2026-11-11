package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/alfacentaur1/dockopt/internal/core"
)

// target is one Dockerfile and what we know about its build context.
type target struct {
	Path       string // as given / found, for display
	ContextDir string
	Input      core.Input
}

// resolve turns a path argument into the Dockerfile path: directories mean
// the Dockerfile inside them.
func resolve(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		path = filepath.Join(path, "Dockerfile")
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
	}
	return path, nil
}

// load reads a Dockerfile together with its .dockerignore and the names of
// the files in its build context.
func load(path, contextDir string) (target, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return target{}, err
	}
	if contextDir == "" {
		contextDir = filepath.Dir(path)
	}
	ignore, err := readIgnore(path, contextDir)
	if err != nil {
		return target{}, err
	}
	return target{
		Path:       path,
		ContextDir: contextDir,
		Input:      core.Input{Dockerfile: src, DockerIgnore: ignore, Files: contextFiles(contextDir)},
	}, nil
}

// readIgnore follows BuildKit: <Dockerfile>.dockerignore next to the
// Dockerfile wins over .dockerignore in the context root. nil means none.
func readIgnore(dockerfile, contextDir string) ([]byte, error) {
	for _, p := range []string{dockerfile + ".dockerignore", filepath.Join(contextDir, ".dockerignore")} {
		b, err := os.ReadFile(p)
		if err == nil {
			if b == nil {
				b = []byte{}
			}
			return b, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return nil, nil
}

// contextFiles lists the names at the root of the build context. That's
// enough to detect the language and which lock files exist.
func contextFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}
