package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/report"
)

// skipDirs are never worth walking into.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".venv": true, "venv": true,
	".terraform": true, ".idea": true, ".vscode": true, "__pycache__": true,
}

// isDockerfile matches Dockerfile, Dockerfile.prod, api.Dockerfile, *.dockerfile.
func isDockerfile(name string) bool {
	low := strings.ToLower(name)
	if strings.HasSuffix(low, ".dockerignore") {
		return false
	}
	return low == "dockerfile" || strings.HasPrefix(low, "dockerfile.") || strings.HasSuffix(low, ".dockerfile")
}

// findDockerfiles walks root and returns every Dockerfile under it.
func findDockerfiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if isDockerfile(d.Name()) {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func (a *app) audit(args []string) (int, error) {
	var c common
	fs := newFlags(a, "audit", "[flags] [dir]")
	c.register(fs, true)
	if err := parse(fs, args); err != nil {
		return 0, err
	}
	opts, err := c.options()
	if err != nil {
		return 0, err
	}
	root := fs.Arg(0)
	if root == "" {
		root = "."
	}

	paths, err := findDockerfiles(root)
	if err != nil {
		return 0, err
	}
	if len(paths) == 0 {
		return 0, fmt.Errorf("no Dockerfiles found under %s", root)
	}

	files := make([]report.File, 0, len(paths))
	for _, p := range paths {
		// each Dockerfile is analyzed against its own directory
		files = append(files, analyzeOne(p, "", opts))
	}
	if err := a.write(a.stdout, c.format, files, c.noColor); err != nil {
		return 0, err
	}
	if c.format == "text" || c.format == "" {
		a.topRules(files)
	}
	return c.exitCode(files)
}

// topRules prints which rules fire most often across the audited files.
func (a *app) topRules(files []report.File) {
	counts := map[string]int{}
	for _, f := range files {
		for _, fd := range f.Findings {
			counts[fd.RuleID]++
		}
	}
	if len(counts) == 0 {
		return
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if counts[ids[i]] != counts[ids[j]] {
			return counts[ids[i]] > counts[ids[j]]
		}
		return ids[i] < ids[j]
	})

	fmt.Fprintln(a.stdout, "\nMost common:")
	tw := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	for _, id := range ids {
		title := ""
		if r := rules.Get(id); r != nil {
			title = r.Title()
		}
		fmt.Fprintf(tw, "  %s\t%d\t%s\n", id, counts[id], title)
	}
	tw.Flush()
}
