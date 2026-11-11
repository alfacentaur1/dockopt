package main

import (
	"fmt"
	"os"

	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/diff"
	"github.com/alfacentaur1/dockopt/internal/report"
)

func (a *app) fix(args []string) (int, error) {
	var c common
	fs := newFlags(a, "fix", "[flags] [Dockerfile|dir ...]")
	c.register(fs, false)
	write := fs.Bool("w", false, "write the fixed Dockerfile back to the file")
	showDiff := fs.Bool("diff", false, "print a unified diff instead of the fixed file")
	check := fs.Bool("check", false, "don't write anything, exit 1 if something would be fixed (for CI)")
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
	// Without -w/-diff/-check the fixed file goes to stdout, which only makes
	// sense for one file.
	plain := !*write && !*showDiff && !*check
	if plain && len(paths) > 1 {
		return 0, fmt.Errorf("fixing several files needs -w, -diff or -check")
	}

	var files []report.File
	changed := false
	for _, p := range paths {
		f, out, err := fixOne(p, c.context, opts)
		if err != nil {
			files = append(files, report.File{Path: p, Error: err.Error()})
			continue
		}
		if len(f.Applied) > 0 {
			changed = true
		}
		switch {
		case plain:
			_, err := a.stdout.Write(out)
			return exitOK, err
		case *write && len(f.Applied) > 0:
			if err := writeKeepingMode(f.Path, out); err != nil {
				return 0, err
			}
		}
		if !*showDiff {
			f.Diff = ""
		}
		files = append(files, f)
	}

	if err := a.write(a.stdout, c.format, files, c.noColor); err != nil {
		return 0, err
	}
	for _, f := range files {
		if f.Error != "" {
			return exitError, nil
		}
	}
	if *check && changed {
		return exitFindings, nil
	}
	return exitOK, nil
}

// fixOne fixes a single Dockerfile and returns the report and the new content.
func fixOne(path, contextDir string, opts core.Options) (report.File, []byte, error) {
	resolved, err := resolve(path)
	if err != nil {
		return report.File{}, nil, err
	}
	t, err := load(resolved, contextDir)
	if err != nil {
		return report.File{}, nil, err
	}
	res, err := core.Fix(t.Input, opts)
	if err != nil {
		return report.File{}, nil, fmt.Errorf("%s: %w", resolved, err)
	}
	return report.File{
		Path:     resolved,
		Lang:     res.Lang,
		Findings: res.Remaining,
		Applied:  res.Applied,
		Diff:     diff.Unified("a/"+resolved, "b/"+resolved, t.Input.Dockerfile, res.Output, 3),
	}, res.Output, nil
}

func writeKeepingMode(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return os.WriteFile(path, data, mode)
}
