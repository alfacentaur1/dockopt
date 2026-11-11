package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alfacentaur1/dockopt/internal/ci"
	"github.com/alfacentaur1/dockopt/internal/report"
)

func (a *app) ci(args []string) (int, error) {
	var c common
	fs := newFlags(a, "ci", "[flags] [Dockerfile|dir ...]")
	c.register(fs, true)
	provider := fs.String("provider", "auto", "where to comment: auto, github, gitlab or none (print the comment)")
	suggest := fs.Bool("suggest", true, "include the suggested fix as a diff")
	if err := parse(fs, args); err != nil {
		return 0, err
	}
	opts, err := c.options()
	if err != nil {
		return 0, err
	}

	paths := fs.Args()
	if len(paths) == 0 {
		if paths, err = findDockerfiles("."); err != nil {
			return 0, err
		}
		if len(paths) == 0 {
			return 0, errors.New("no Dockerfiles found")
		}
	}

	files := make([]report.File, 0, len(paths))
	for _, p := range paths {
		f := analyzeOne(p, c.context, opts)
		if f.Error == "" && *suggest {
			if fixed, _, err := fixOne(p, c.context, opts); err == nil && len(fixed.Applied) > 0 {
				f.Diff = fixed.Diff
			}
		}
		files = append(files, f)
	}

	var body bytes.Buffer
	if err := report.Markdown(&body, files); err != nil {
		return 0, err
	}

	var p ci.Provider
	switch *provider {
	case "none":
	case "auto":
		p, err = ci.Detect(os.Getenv, nil)
	default:
		p, err = ci.ByName(*provider, os.Getenv, nil)
	}
	switch {
	case errors.Is(err, ci.ErrNotInMergeRequest):
		fmt.Fprintln(a.stderr, "dockopt: not a merge request pipeline, printing the comment instead")
		p = nil
	case err != nil:
		return 0, err
	}

	if p == nil {
		fmt.Fprint(a.stdout, body.String())
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := p.Upsert(ctx, body.String()); err != nil {
			return 0, fmt.Errorf("posting the %s comment: %w", p.Name(), err)
		}
		if err := a.write(a.stdout, "text", files, c.noColor); err != nil {
			return 0, err
		}
		fmt.Fprintf(a.stdout, "commented on the %s merge request\n", p.Name())
	}
	return c.exitCode(files)
}
