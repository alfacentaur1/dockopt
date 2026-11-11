package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/alfacentaur1/dockopt/internal/bench"
	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/report"
)

func (a *app) bench(args []string) (int, error) {
	var c common
	fs := newFlags(a, "bench", "[flags] [Dockerfile|dir]")
	c.register(fs, false)
	touch := fs.String("touch", "", "file (relative to the context) to modify before the warm build; default: a source file")
	trivy := fs.Bool("trivy", false, "also count CVEs with trivy")
	runs := fs.Int("runs", 1, "repeat every build this many times and report the median")
	timeout := fs.Duration("timeout", 30*time.Minute, "give up after this long")
	if err := parse(fs, args); err != nil {
		return 0, err
	}
	opts, err := c.options()
	if err != nil {
		return 0, err
	}

	path := fs.Arg(0)
	if path == "" {
		path = "Dockerfile"
	}
	resolved, err := resolve(path)
	if err != nil {
		return 0, err
	}
	t, err := load(resolved, c.context)
	if err != nil {
		return 0, err
	}
	fixed, err := core.Fix(t.Input, opts)
	if err != nil {
		return 0, err
	}
	if !fixed.Changed() {
		fmt.Fprintf(a.stdout, "%s: nothing to fix, so nothing to compare\n", resolved)
		return exitOK, nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, *timeout)
	defer cancelTimeout()

	docker, err := bench.NewDocker(ctx)
	if err != nil {
		return 0, err
	}
	defer docker.Close()
	var scanner bench.Scanner
	if *trivy {
		tv, err := bench.NewTrivy()
		if err != nil {
			return 0, err
		}
		scanner = tv
	}

	res, err := bench.Run(ctx, bench.Options{
		ContextDir: t.ContextDir,
		Original:   t.Input.Dockerfile,
		Fixed:      fixed.Output,
		Touch:      *touch,
		Runs:       *runs,
		Log:        a.stderr,
	}, docker, scanner)
	if err != nil {
		return 0, err
	}
	return exitOK, report.Bench(a.stdout, c.format, report.BenchReport{
		Path: resolved, Fixes: len(fixed.Applied), Result: res, Touched: res.Touched,
	})
}
