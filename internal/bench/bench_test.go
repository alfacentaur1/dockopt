package bench

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeEngine struct {
	pulled  []string
	builds  []string // "tag nocache=bool"
	removed []string
	info    map[string]ImageInfo // by tag prefix
	failOn  string
	seen    []string // Dockerfile content at build time
}

func (f *fakeEngine) Pull(_ context.Context, img string) error {
	f.pulled = append(f.pulled, img)
	return nil
}

func (f *fakeEngine) Build(_ context.Context, dir, dockerfile, tag string, noCache bool) error {
	if f.failOn != "" && strings.Contains(tag, f.failOn) {
		return errors.New("build failed")
	}
	b, err := os.ReadFile(dockerfile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		return errors.New("context was not copied")
	}
	f.seen = append(f.seen, string(b))
	f.builds = append(f.builds, strings.SplitN(tag, ":", 2)[0]+" nocache="+map[bool]string{true: "true", false: "false"}[noCache])
	return nil
}

func (f *fakeEngine) Inspect(_ context.Context, tag string) (ImageInfo, error) {
	for prefix, info := range f.info {
		if strings.HasPrefix(tag, prefix) {
			return info, nil
		}
	}
	return ImageInfo{}, errors.New("no such image")
}

func (f *fakeEngine) Remove(_ context.Context, tag string) error {
	f.removed = append(f.removed, strings.SplitN(tag, ":", 2)[0])
	return nil
}

type fakeScanner struct{ n int }

func (s fakeScanner) CVEs(context.Context, string) (int, error) { return s.n, nil }

func goProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"main.go":         "package main\n",
		"go.mod":          "module x\n",
		"internal/a.go":   "package internal\n",
		".git/HEAD":       "ref: main\n",
		"static/site.css": "body{}\n",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	return dir
}

func TestRun(t *testing.T) {
	eng := &fakeEngine{info: map[string]ImageInfo{
		"dockopt-bench-original": {Size: 900, Layers: 9},
		"dockopt-bench-fixed":    {Size: 300, Layers: 5},
	}}
	opts := Options{
		ContextDir: goProject(t),
		Original:   []byte("FROM golang:1.22 AS build\nCOPY . .\nFROM build\n"),
		Fixed:      []byte("FROM golang:1.22 AS build\nCOPY go.mod ./\nFROM gcr.io/distroless/static\n"),
	}
	res, err := Run(context.Background(), opts, eng, fakeScanner{n: 3})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"golang:1.22", "gcr.io/distroless/static"}; !reflect.DeepEqual(eng.pulled, want) {
		t.Errorf("pulled %v, want %v", eng.pulled, want)
	}
	wantBuilds := []string{
		"dockopt-bench-original nocache=true", "dockopt-bench-original nocache=false",
		"dockopt-bench-fixed nocache=true", "dockopt-bench-fixed nocache=false",
	}
	if !reflect.DeepEqual(eng.builds, wantBuilds) {
		t.Errorf("builds %v", eng.builds)
	}
	if eng.seen[0] != string(opts.Original) || eng.seen[2] != string(opts.Fixed) {
		t.Error("each variant must build its own Dockerfile")
	}
	if len(eng.removed) != 2 {
		t.Errorf("images not cleaned up: %v", eng.removed)
	}
	if res.Original.Size != 900 || res.Fixed.Layers != 5 || *res.Fixed.CVEs != 3 {
		t.Errorf("unexpected result %+v", res)
	}
	if res.Touched != "main.go" {
		t.Errorf("touched %q, want main.go", res.Touched)
	}
	// the original context must not be modified by touching
	if b, _ := os.ReadFile(filepath.Join(opts.ContextDir, "main.go")); string(b) != "package main\n" {
		t.Error("bench modified the real project")
	}
}

func TestRunRepeatsBuilds(t *testing.T) {
	eng := &fakeEngine{info: map[string]ImageInfo{"dockopt-bench": {}}}
	res, err := Run(context.Background(), Options{
		ContextDir: goProject(t), Original: []byte("FROM alpine:3.20\n"), Fixed: []byte("FROM alpine:3.20\n"), Runs: 3,
	}, eng, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(eng.builds) != 12 || res.Runs != 3 {
		t.Fatalf("3 cold + 3 warm per variant expected, got %d builds, runs=%d", len(eng.builds), res.Runs)
	}
}

func TestMedian(t *testing.T) {
	if got := median([]time.Duration{5, 1, 3}); got != 3 {
		t.Errorf("odd: got %v", got)
	}
	if got := median([]time.Duration{4, 1, 3, 2}); got != 2 {
		t.Errorf("even: got %v", got)
	}
}

func TestRunBuildFailure(t *testing.T) {
	eng := &fakeEngine{failOn: "fixed", info: map[string]ImageInfo{"dockopt-bench": {}}}
	_, err := Run(context.Background(), Options{
		ContextDir: goProject(t), Original: []byte("FROM alpine:3.20\n"), Fixed: []byte("FROM alpine:3.20\n"),
	}, eng, nil)
	if err == nil || !strings.Contains(err.Error(), "fixed") {
		t.Fatalf("expected a fixed-variant build error, got %v", err)
	}
}

func TestPickSource(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"go.mod", "Dockerfile", "web/app.js", "node_modules/x/index.js", "README.md"} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, nil, 0o644)
	}
	got, err := pickSource(dir)
	if err != nil || got != "web/app.js" {
		t.Fatalf("got %q, %v", got, err)
	}

	empty := t.TempDir()
	os.WriteFile(filepath.Join(empty, "README.md"), nil, 0o644)
	if _, err := pickSource(empty); !errors.Is(err, errNothingToTouch) {
		t.Fatalf("expected errNothingToTouch, got %v", err)
	}
}

func TestBaseImages(t *testing.T) {
	got := baseImages(
		[]byte("ARG V\nFROM golang:${V} AS a\nFROM golang:1.22 AS build\nFROM build\nFROM scratch\n"),
		[]byte("FROM golang:1.22\nFROM alpine:3.20\n"),
	)
	if want := []string{"golang:1.22", "alpine:3.20"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
