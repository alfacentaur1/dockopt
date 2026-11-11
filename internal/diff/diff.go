// Package diff produces unified diffs. Dockerfiles are small, so a plain
// LCS table is plenty and keeps this dependency free (it also runs in wasm).
package diff

import (
	"fmt"
	"strings"
)

type op struct {
	kind byte // ' ', '-', '+'
	text string
}

// Unified returns a unified diff of a and b with the given number of
// context lines, or "" when they are equal.
func Unified(nameA, nameB string, a, b []byte, context int) string {
	if string(a) == string(b) {
		return ""
	}
	ops := edits(lines(string(a)), lines(string(b)))

	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", nameA, nameB)

	// walk the edit script and cut it into hunks
	for i := 0; i < len(ops); {
		if ops[i].kind == ' ' {
			i++
			continue
		}
		start := max(0, i-context)
		end := i
		for end < len(ops) {
			if ops[end].kind != ' ' {
				end++
				continue
			}
			// a run of unchanged lines; does another change follow closely?
			run := end
			for run < len(ops) && ops[run].kind == ' ' {
				run++
			}
			if run == len(ops) || run-end > 2*context {
				end = min(len(ops), end+context)
				break
			}
			end = run
		}
		writeHunk(&out, ops, start, end)
		i = end
	}
	return out.String()
}

func writeHunk(out *strings.Builder, ops []op, start, end int) {
	// line numbers of the hunk start in a and b (1-based)
	aLine, bLine := 1, 1
	for _, o := range ops[:start] {
		if o.kind != '+' {
			aLine++
		}
		if o.kind != '-' {
			bLine++
		}
	}
	aCount, bCount := 0, 0
	for _, o := range ops[start:end] {
		if o.kind != '+' {
			aCount++
		}
		if o.kind != '-' {
			bCount++
		}
	}
	if aCount == 0 {
		aLine--
	}
	if bCount == 0 {
		bLine--
	}
	fmt.Fprintf(out, "@@ -%d,%d +%d,%d @@\n", aLine, aCount, bLine, bCount)
	for _, o := range ops[start:end] {
		out.WriteByte(o.kind)
		out.WriteString(o.text)
		out.WriteByte('\n')
	}
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// edits computes an edit script from a longest common subsequence table.
func edits(a, b []string) []op {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	var out []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, op{' ', a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, op{'-', a[i]})
			i++
		default:
			out = append(out, op{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, op{'-', a[i]})
	}
	for ; j < m; j++ {
		out = append(out, op{'+', b[j]})
	}
	return out
}
