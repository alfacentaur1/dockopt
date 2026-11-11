package main

import (
	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/report"
)

func (a *app) analyze(args []string) (int, error) {
	var c common
	fs := newFlags(a, "analyze", "[flags] [Dockerfile|dir ...]")
	c.register(fs, true)
	if err := parse(fs, args); err != nil {
		return 0, err
	}
	opts, err := c.options()
	if err != nil {
		return 0, err
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"Dockerfile"}
	}
	files := make([]report.File, 0, len(paths))
	for _, p := range paths {
		files = append(files, analyzeOne(p, c.context, opts))
	}
	if err := a.write(a.stdout, c.format, files, c.noColor); err != nil {
		return 0, err
	}
	return c.exitCode(files)
}

func analyzeOne(path, contextDir string, opts core.Options) report.File {
	resolved, err := resolve(path)
	if err != nil {
		return report.File{Path: path, Error: err.Error()}
	}
	t, err := load(resolved, contextDir)
	if err != nil {
		return report.File{Path: resolved, Error: err.Error()}
	}
	res, err := core.Analyze(t.Input, opts)
	if err != nil {
		return report.File{Path: resolved, Error: err.Error()}
	}
	return report.File{Path: resolved, Lang: res.Lang, Findings: res.Findings}
}
