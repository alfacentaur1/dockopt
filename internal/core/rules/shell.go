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

var commandEnd = regexp.MustCompile(`&&|\|\||;|\|`)

// insertFlag adds flag right after every match of re (e.g. `apt-get install`)
// whose command doesn't already contain the flag. Works on raw text, so
// line continuations and indentation survive.
func insertFlag(raw string, re *regexp.Regexp, flag string) (string, bool) {
	matches := re.FindAllStringIndex(raw, -1)
	changed := false
	for i := len(matches) - 1; i >= 0; i-- {
		end := matches[i][1]
		rest := raw[end:]
		if loc := commandEnd.FindStringIndex(rest); loc != nil {
			rest = rest[:loc[0]]
		}
		if strings.Contains(rest, flag) {
			continue
		}
		raw = raw[:end] + " " + flag + raw[end:]
		changed = true
	}
	return raw, changed
}

// missingFlag reports whether some match of re in cmd lacks flag in its command.
func missingFlag(cmd string, re *regexp.Regexp, flag string) bool {
	_, changed := insertFlag(cmd, re, flag)
	return changed
}

// replaceKeyword swaps the instruction keyword at the start of raw.
func replaceKeyword(raw, keyword string) string {
	trimmed := strings.TrimLeft(raw, " \t")
	lead := raw[:len(raw)-len(trimmed)]
	if i := strings.IndexAny(trimmed, " \t"); i > 0 {
		return lead + keyword + trimmed[i:]
	}
	return lead + keyword
}
