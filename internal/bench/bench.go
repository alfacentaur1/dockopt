// Package bench is the only part of dockopt that talks to docker. It builds
// the original and the fixed Dockerfile and measures what changed:
//
//   - final image size and layer count
//   - cold build time (no cache)
//   - warm build time after touching one source file; this is the number
//     that shows whether cache ordering actually worked
//   - optionally the CVE count via trivy
package bench

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/alfacentaur1/dockopt/internal/core"
)

// Engine builds and inspects images. The real one is docker; tests use a fake.
type Engine interface {
	Pull(ctx context.Context, image string) error
	Build(ctx context.Context, contextDir, dockerfile, tag string, noCache bool) error
	Inspect(ctx context.Context, tag string) (ImageInfo, error)
	Remove(ctx context.Context, tag string) error
}

// Scanner counts vulnerabilities in an image.
type Scanner interface {
	CVEs(ctx context.Context, tag string) (int, error)
}

// ImageInfo is what we read back from a built image.
type ImageInfo struct {
	Size   int64
	Layers int
}

// Measurement is the result for one Dockerfile.
type Measurement struct {
	Size   int64         `json:"size_bytes"`
	Layers int           `json:"layers"`
	Cold   time.Duration `json:"cold_build_ns"`
	Warm   time.Duration `json:"warm_build_ns"`
	CVEs   *int          `json:"cves,omitempty"` // nil when not scanned
}

// Result compares the original and the fixed Dockerfile.
type Result struct {
	Original Measurement `json:"original"`
	Fixed    Measurement `json:"fixed"`
	Touched  string      `json:"touched"` // file modified before the warm build
	Runs     int         `json:"runs"`    // build times are the median of this many builds
}

// Options for Run.
type Options struct {
	ContextDir string
	Original   []byte
	Fixed      []byte
	// Touch is the file (relative to ContextDir) modified before the warm
	// build. Empty picks a source file automatically.
	Touch string
	// Runs repeats every build and keeps the median, because a single build
	// time is noisy. Values below 1 mean 1.
	Runs int
	// Log receives progress lines; may be nil.
	Log io.Writer
}

// Run builds both variants and measures them. Base images are pulled first so
// that downloading them doesn't count against whichever variant goes first.
func Run(ctx context.Context, opts Options, eng Engine, scan Scanner) (*Result, error) {
	logf := func(format string, args ...any) {
		if opts.Log != nil {
			fmt.Fprintf(opts.Log, format+"\n", args...)
		}
	}

	for _, img := range baseImages(opts.Original, opts.Fixed) {
		logf("pulling %s", img)
		if err := eng.Pull(ctx, img); err != nil {
			return nil, fmt.Errorf("pull %s: %w", img, err)
		}
	}

	res := &Result{Runs: max(1, opts.Runs)}
	for _, v := range []struct {
		name string
		src  []byte
		dst  *Measurement
	}{{"original", opts.Original, &res.Original}, {"fixed", opts.Fixed, &res.Fixed}} {
		m, touched, err := measure(ctx, v.name, v.src, opts, eng, scan, logf)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", v.name, err)
		}
		*v.dst = m
		res.Touched = touched
	}
	return res, nil
}

func measure(ctx context.Context, name string, src []byte, opts Options, eng Engine, scan Scanner,
	logf func(string, ...any)) (Measurement, string, error) {

	dir, err := os.MkdirTemp("", "dockopt-bench-")
	if err != nil {
		return Measurement{}, "", err
	}
	defer os.RemoveAll(dir)
	if err := copyContext(opts.ContextDir, dir); err != nil {
		return Measurement{}, "", fmt.Errorf("copy build context: %w", err)
	}
	dockerfile := filepath.Join(dir, "Dockerfile.dockopt-bench")
	if err := os.WriteFile(dockerfile, src, 0o644); err != nil {
		return Measurement{}, "", err
	}

	tag := "dockopt-bench-" + name + ":" + randomTag()
	defer eng.Remove(context.WithoutCancel(ctx), tag) //nolint:errcheck // best effort cleanup

	runs := max(1, opts.Runs)
	timed := func(noCache bool) (time.Duration, error) {
		start := time.Now()
		err := eng.Build(ctx, dir, dockerfile, tag, noCache)
		return time.Since(start), err
	}

	var m Measurement
	cold := make([]time.Duration, 0, runs)
	for i := 0; i < runs; i++ {
		logf("%s: cold build %d/%d", name, i+1, runs)
		d, err := timed(true)
		if err != nil {
			return m, "", err
		}
		cold = append(cold, d)
	}
	m.Cold = median(cold)

	info, err := eng.Inspect(ctx, tag)
	if err != nil {
		return m, "", err
	}
	m.Size, m.Layers = info.Size, info.Layers

	var touched string
	warm := make([]time.Duration, 0, runs)
	for i := 0; i < runs; i++ {
		if touched, err = touch(dir, opts.Touch); err != nil {
			return m, "", err
		}
		logf("%s: warm build %d/%d after touching %s", name, i+1, runs, touched)
		d, err := timed(false)
		if err != nil {
			return m, "", err
		}
		warm = append(warm, d)
	}
	m.Warm = median(warm)

	if scan != nil {
		logf("%s: scanning for CVEs", name)
		n, err := scan.CVEs(ctx, tag)
		if err != nil {
			return m, "", fmt.Errorf("trivy: %w", err)
		}
		m.CVEs = &n
	}
	return m, touched, nil
}

func median(ds []time.Duration) time.Duration {
	s := slices.Clone(ds)
	slices.Sort(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

// baseImages lists external images used by FROM in either Dockerfile.
func baseImages(srcs ...[]byte) []string {
	seen := map[string]bool{}
	var out []string
	for _, src := range srcs {
		df, err := core.Parse(src)
		if err != nil {
			continue
		}
		stages := map[string]bool{}
		for _, st := range df.Stages() {
			img := st.BaseImage
			low := strings.ToLower(img)
			if low != "scratch" && !strings.Contains(img, "$") && !stages[low] && !seen[img] {
				seen[img] = true
				out = append(out, img)
			}
			if st.Name != "" {
				stages[strings.ToLower(st.Name)] = true
			}
		}
	}
	return out
}

func randomTag() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// copyContext copies the build context, minus .git, into dst.
func copyContext(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case d.Type().IsRegular():
			return copyFile(path, target)
		}
		return nil
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// sourceExt is the order in which source files are preferred for touching.
var sourceExt = []string{".go", ".ts", ".js", ".py", ".java", ".kt", ".rb", ".rs", ".php", ".cs", ".c", ".cpp", ".html", ".css"}

// notSource are files whose change is supposed to invalidate dependencies.
var notSource = map[string]bool{
	"go.mod": true, "go.sum": true, "package.json": true, "package-lock.json": true, "yarn.lock": true,
	"pnpm-lock.yaml": true, "requirements.txt": true, "pyproject.toml": true, "poetry.lock": true,
	"Pipfile": true, "Pipfile.lock": true, "pom.xml": true, "build.gradle": true, ".dockerignore": true,
}

var errNothingToTouch = errors.New("found no source file to touch before the warm build; pass --touch <file>")

// touch appends a newline to a source file, the smallest change that still
// invalidates every layer that copied it.
func touch(dir, rel string) (string, error) {
	if rel == "" {
		var err error
		if rel, err = pickSource(dir); err != nil {
			return "", err
		}
	}
	f, err := os.OpenFile(filepath.Join(dir, filepath.FromSlash(rel)), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString("\n"); err != nil {
		f.Close()
		return "", err
	}
	return rel, f.Close()
}

// pickSource prefers the most typical source extension, then files closer to
// the root (main.go over internal/x/y.go).
func pickSource(dir string) (string, error) {
	best, bestRank, bestDepth := "", len(sourceExt), 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", ".venv":
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if notSource[name] || strings.HasPrefix(name, "Dockerfile") {
			return nil
		}
		for rank, ext := range sourceExt {
			if !strings.HasSuffix(name, ext) {
				continue
			}
			rel, _ := filepath.Rel(dir, path)
			rel = filepath.ToSlash(rel)
			depth := strings.Count(rel, "/")
			if rank < bestRank || (rank == bestRank && depth < bestDepth) {
				best, bestRank, bestDepth = rel, rank, depth
			}
			break
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if best == "" {
		return "", errNothingToTouch
	}
	return best, nil
}
