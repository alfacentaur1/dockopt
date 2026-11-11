package rules

import (
	"regexp"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

// Helpers for looking inside RUN commands. This is deliberately not a real
// shell parser: Dockerfiles are full of `a && b \` chains, and a couple of
// regexes over the normalized command get the job done.

var spaces = regexp.MustCompile(`\s+`)

// command returns the command of a RUN/CMD/ENTRYPOINT, whitespace collapsed.
func command(in *model.Instruction) string {
	return normalize(strings.Join(in.Args, " "))
}

// normalize collapses whitespace and line continuations.
func normalize(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")
	return strings.TrimSpace(spaces.ReplaceAllString(cmd, " "))
}

var separators = regexp.MustCompile(`\s*(?:&&|\|\||;|\|)\s*`)

// commands splits a shell command into simple commands on &&, ||, ; and |.
// Quotes are ignored, which is fine for the patterns rules look for.
func commands(cmd string) []string {
	var out []string
	for _, c := range separators.Split(normalize(cmd), -1) {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// insertAfter inserts text right after the first match of re in raw.
func insertAfter(raw string, re *regexp.Regexp, text string) (string, bool) {
	loc := re.FindStringIndex(raw)
	if loc == nil {
		return raw, false
	}
	return raw[:loc[1]] + " " + text + raw[loc[1]:], true
}

// appendCommand adds `&& cmd` at the end of a RUN. Multi-line RUNs get the
// new command on its own continuation line, indented like the others.
func appendCommand(raw, cmd string) string {
	trimmed := strings.TrimRight(raw, " \t")
	if !strings.Contains(trimmed, "\n") {
		return trimmed + " && " + cmd
	}
	return trimmed + " \\\n" + continuationIndent(trimmed) + "&& " + cmd
}

// continuationIndent is the leading whitespace of the last line of raw.
func continuationIndent(raw string) string {
	last := raw[strings.LastIndex(raw, "\n")+1:]
	indent := last[:len(last)-len(strings.TrimLeft(last, " \t"))]
	if indent == "" {
		return "    "
	}
	return indent
}

// hasCacheMount reports whether a RUN mounts a cache over path, in which
// case cleaning up that path inside the layer is pointless.
func hasCacheMount(in *model.Instruction, path string) bool {
	for _, f := range in.Flags {
		if strings.HasPrefix(f, "--mount=") && strings.Contains(f, "type=cache") && strings.Contains(f, path) {
			return true
		}
	}
	return false
}

// firstWord returns the instruction keyword as written in raw (FROM, from...).
func replaceKeyword(raw, keyword string) string {
	trimmed := strings.TrimLeft(raw, " \t")
	lead := raw[:len(raw)-len(trimmed)]
	if i := strings.IndexAny(trimmed, " \t"); i > 0 {
		return lead + keyword + trimmed[i:]
	}
	return lead + keyword
}
