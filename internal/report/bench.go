package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/alfacentaur1/dockopt/internal/bench"
)

// BenchReport is a bench result with the context needed to print it.
type BenchReport struct {
	Path    string        `json:"path"`
	Fixes   int           `json:"fixes_applied"`
	Result  *bench.Result `json:"result"`
	Touched string        `json:"touched"`
}

type benchRow struct {
	name            string
	original, fixed string
	change          string
}

func benchRows(r *bench.Result) []benchRow {
	o, f := r.Original, r.Fixed
	rows := []benchRow{
		{"image size", humanBytes(o.Size), humanBytes(f.Size), percent(float64(o.Size), float64(f.Size))},
		{"layers", fmt.Sprint(o.Layers), fmt.Sprint(f.Layers), signed(f.Layers - o.Layers)},
		{"cold build", humanDuration(o.Cold), humanDuration(f.Cold), percent(float64(o.Cold), float64(f.Cold))},
		{"warm build", humanDuration(o.Warm), humanDuration(f.Warm), percent(float64(o.Warm), float64(f.Warm))},
	}
	if o.CVEs != nil && f.CVEs != nil {
		rows = append(rows, benchRow{"CVEs", fmt.Sprint(*o.CVEs), fmt.Sprint(*f.CVEs), signed(*f.CVEs - *o.CVEs)})
	}
	return rows
}

// Bench prints a bench comparison in the given format.
func Bench(w io.Writer, format string, b BenchReport) error {
	switch format {
	case "", "text":
		fmt.Fprintf(w, "%s (%d fixes applied)\n\n", b.Path, b.Fixes)
		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.AlignRight)
		fmt.Fprintln(tw, "\toriginal\tfixed\tchange\t")
		for _, r := range benchRows(b.Result) {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", r.name, r.original, r.fixed, r.change)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		fmt.Fprintf(w, "\nwarm build = rebuild after touching %s%s\n", b.Touched, runsNote(b.Result))
		return nil
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(b)
	case "md", "markdown":
		fmt.Fprintf(w, "### dockopt bench: `%s`\n\n%d fixes applied.\n\n", b.Path, b.Fixes)
		fmt.Fprintln(w, "| | original | fixed | change |")
		fmt.Fprintln(w, "|---|---:|---:|---:|")
		for _, r := range benchRows(b.Result) {
			fmt.Fprintf(w, "| %s | %s | %s | %s |\n", r.name, r.original, r.fixed, r.change)
		}
		fmt.Fprintf(w, "\nWarm build = rebuild after touching `%s`%s.\n", b.Touched, runsNote(b.Result))
		return nil
	}
	return fmt.Errorf("unknown format %q", format)
}

func runsNote(r *bench.Result) string {
	if r.Runs > 1 {
		return fmt.Sprintf("; build times are the median of %d runs", r.Runs)
	}
	return ""
}

// humanBytes uses decimal units, like docker images does.
func humanBytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}

func humanDuration(d time.Duration) string {
	if d >= time.Minute {
		return d.Round(time.Second).String()
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func percent(before, after float64) string {
	if before == 0 {
		return "-"
	}
	return fmt.Sprintf("%+.1f%%", (after-before)/before*100)
}

func signed(n int) string {
	if n == 0 {
		return "0"
	}
	return fmt.Sprintf("%+d", n)
}
