package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

func (a *app) rules(args []string) (int, error) {
	fs := newFlags(a, "rules", "[flags] [RULE_ID]")
	format := fs.String("format", "text", "output format: text, json or md")
	if err := parse(fs, args); err != nil {
		return 0, err
	}

	if id := fs.Arg(0); id != "" {
		r := rules.Get(id)
		if r == nil {
			return 0, fmt.Errorf("unknown rule %q", id)
		}
		fix := "no"
		if rules.IsFixable(r) {
			fix = "yes"
		}
		fmt.Fprintf(a.stdout, "%s  %s\n\nseverity: %s\nfixable:  %s\n\n%s\n", r.ID(), r.Title(), r.Severity(), fix, r.Description())
		return exitOK, nil
	}

	all := rules.All()
	switch *format {
	case "json":
		type row struct {
			ID          string         `json:"id"`
			Title       string         `json:"title"`
			Severity    rules.Severity `json:"severity"`
			Fixable     bool           `json:"fixable"`
			Description string         `json:"description"`
		}
		rows := make([]row, len(all))
		for i, r := range all {
			rows[i] = row{r.ID(), r.Title(), r.Severity(), rules.IsFixable(r), r.Description()}
		}
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		return exitOK, enc.Encode(rows)
	case "md":
		fmt.Fprintln(a.stdout, "| ID | Severity | Fix | Rule |")
		fmt.Fprintln(a.stdout, "|---|---|:-:|---|")
		for _, r := range all {
			fix := ""
			if rules.IsFixable(r) {
				fix = "✓"
			}
			fmt.Fprintf(a.stdout, "| `%s` | %s | %s | %s |\n", r.ID(), r.Severity(), fix, r.Title())
		}
		return exitOK, nil
	case "text":
		tw := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tSEVERITY\tFIX\tRULE")
		for _, r := range all {
			fix := "-"
			if rules.IsFixable(r) {
				fix = "yes"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.ID(), r.Severity(), fix, r.Title())
		}
		return exitOK, tw.Flush()
	}
	return 0, fmt.Errorf("unknown format %q", *format)
}
