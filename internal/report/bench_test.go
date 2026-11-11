package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/alfacentaur1/dockopt/internal/bench"
)

func TestBenchText(t *testing.T) {
	cves := func(n int) *int { return &n }
	b := BenchReport{Path: "Dockerfile", Fixes: 4, Touched: "main.go", Result: &bench.Result{
		Original: bench.Measurement{Size: 912_300_000, Layers: 12, Cold: 62 * time.Second, Warm: 58 * time.Second, CVEs: cves(143)},
		Fixed:    bench.Measurement{Size: 18_400_000, Layers: 6, Cold: 41200 * time.Millisecond, Warm: 3400 * time.Millisecond, CVEs: cves(12)},
	}}

	var out bytes.Buffer
	if err := Bench(&out, "text", b); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"912.3 MB", "18.4 MB", "-98.0%", "1m2s", "3.4s", "-94.1%", "-131", "-6", "touching main.go"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}

	out.Reset()
	if err := Bench(&out, "md", b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "| warm build | 58.0s | 3.4s | -94.1% |") {
		t.Errorf("unexpected markdown:\n%s", out.String())
	}
}

func TestHumanBytes(t *testing.T) {
	for n, want := range map[int64]string{999: "999 B", 1000: "1.0 kB", 1_500_000: "1.5 MB", 2_000_000_000: "2.0 GB"} {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
