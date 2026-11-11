// Package render writes a model.Dockerfile back to text.
//
// Untouched instructions are copied from the original source byte for byte,
// together with the comments and blank lines around them. Only instructions a
// fixer changed or added are re-rendered, which keeps diffs small.
package render

import (
	"encoding/json"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

// Render returns the Dockerfile as text, always ending with a newline.
func Render(df *model.Dockerfile) []byte {
	var b strings.Builder
	cursor := 0 // last source line already written, 1-based

	writeSource := func(from, to int) {
		for l := from; l <= to && l <= len(df.Source); l++ {
			b.WriteString(df.Source[l-1])
			b.WriteByte('\n')
		}
	}

	for _, in := range df.Instructions {
		if in.StartLine > 0 {
			// comments and blank lines between the previous instruction and this one
			writeSource(cursor+1, in.StartLine-1)
			if in.EndLine > cursor {
				cursor = in.EndLine
			}
		}
		switch {
		case in.Deleted:
		case in.StartLine > 0 && !in.Dirty:
			writeSource(in.StartLine, in.EndLine)
		default:
			b.WriteString(Instruction(in))
			b.WriteByte('\n')
		}
	}
	writeSource(cursor+1, len(df.Source))
	return []byte(b.String())
}

// Instruction renders a single instruction. Raw text wins when a fixer set it.
func Instruction(in *model.Instruction) string {
	if in.Raw != "" {
		return in.Raw
	}

	parts := []string{in.Cmd}
	parts = append(parts, in.Flags...)
	if in.JSON {
		parts = append(parts, execForm(in.Args))
	} else {
		parts = append(parts, in.Args...)
	}
	return strings.Join(parts, " ")
}

// execForm renders args as a JSON array: ["a", "b"].
func execForm(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		q, _ := json.Marshal(a)
		quoted[i] = string(q)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// ExecForm is execForm for callers outside the package (fixers converting
// shell form to exec form).
func ExecForm(args []string) string { return execForm(args) }
