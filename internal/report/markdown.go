package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
)

// Markdown writes a report suitable for a merge request comment or a file.
func Markdown(w io.Writer, files []File) error {
	s := Summarize(files)
	fmt.Fprintf(w, "### dockopt\n\n%s\n", summaryLine(s))

	for _, f := range files {
		fmt.Fprintf(w, "\n#### `%s`\n\n", f.Path)
		if f.Error != "" {
			fmt.Fprintf(w, "Couldn't analyze: %s\n", f.Error)
			continue
		}
		if len(f.Findings) == 0 {
			fmt.Fprintln(w, "No issues.")
		} else {
			fmt.Fprintln(w, "| Line | Severity | Rule | Message |")
			fmt.Fprintln(w, "|---:|---|---|---|")
			for _, fd := range f.Findings {
				msg := escapeCell(fd.Message)
				if fd.Fixable {
					msg += " _(fixable)_"
				}
				fmt.Fprintf(w, "| %d | %s | `%s` | %s |\n", fd.Line, icon(fd.Severity), fd.RuleID, msg)
			}
		}
		if f.Diff != "" {
			fmt.Fprintf(w, "\n<details><summary>Suggested fix</summary>\n\n```diff\n%s```\n\n</details>\n", f.Diff)
		}
	}
	return nil
}

func icon(s rules.Severity) string {
	switch s {
	case rules.Error:
		return "🔴 error"
	case rules.Warning:
		return "🟡 warning"
	default:
		return "🔵 info"
	}
}

func escapeCell(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}
