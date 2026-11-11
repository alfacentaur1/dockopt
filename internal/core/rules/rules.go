// Package rules holds every check dockopt knows about.
//
// One file per rule. A rule implements Rule; if it can fix what it finds it
// also implements Fixer. Rules register themselves in init().
//
// Rule IDs are stable (DO001...). Once an ID is published it is never reused,
// because people put them in ignore lists.
package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

// Severity of a finding.
type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "info"
	}
}

// MarshalText makes severities show up as words in JSON.
func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// ParseSeverity accepts info, warning (or warn) and error.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info":
		return Info, nil
	case "warning", "warn":
		return Warning, nil
	case "error":
		return Error, nil
	}
	return Info, fmt.Errorf("unknown severity %q (use info, warning or error)", s)
}

// Finding is one problem in one place.
type Finding struct {
	RuleID   string   `json:"rule"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
	Fixable  bool     `json:"fixable"`

	// Instruction the finding points at, for the fixer. Not serialized.
	Instruction *model.Instruction `json:"-"`
}

// Context is what a rule gets: the parsed file plus what we know about the
// build context around it. Rules need more than the AST, e.g. whether this is
// a Go or a Node project, or whether a .dockerignore exists.
type Context struct {
	File            *model.Dockerfile
	DockerIgnore    []string // patterns, comments and blank lines removed
	HasDockerIgnore bool
	Files           []string // file names in the build context, relative, slash separated
	Lang            string   // go, node, python, java or ""
}

// HasFile reports whether the build context contains a file with this name
// at its root.
func (c *Context) HasFile(name string) bool {
	for _, f := range c.Files {
		if f == name {
			return true
		}
	}
	return false
}

// Rule is a single check.
type Rule interface {
	ID() string
	Title() string
	Description() string
	Severity() Severity
	Check(ctx *Context) []Finding
}

// Fixer is implemented by rules that can fix their findings. Fix mutates
// ctx.File and returns true when it changed something. It must not touch
// the file when it returns false.
type Fixer interface {
	Fix(ctx *Context, f Finding) bool
}

var registry = map[string]Rule{}

// Register adds a rule. It panics on duplicate IDs: that's a programming
// error, not something to recover from.
func Register(r Rule) {
	if _, dup := registry[r.ID()]; dup {
		panic("rules: duplicate rule id " + r.ID())
	}
	registry[r.ID()] = r
}

// All returns every registered rule ordered by ID.
func All() []Rule {
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Get returns the rule with the given ID, or nil.
func Get(id string) Rule { return registry[strings.ToUpper(id)] }

// IsFixable reports whether the rule implements Fixer.
func IsFixable(r Rule) bool {
	_, ok := r.(Fixer)
	return ok
}

// meta implements the boring half of Rule so rule files only contain logic.
type meta struct {
	id, title, desc string
	sev             Severity
}

func (m meta) ID() string          { return m.id }
func (m meta) Title() string       { return m.title }
func (m meta) Description() string { return m.desc }
func (m meta) Severity() Severity  { return m.sev }

func (m meta) finding(in *model.Instruction, fixable bool, format string, args ...any) Finding {
	return Finding{
		RuleID:      m.id,
		Severity:    m.sev,
		Line:        in.StartLine,
		Message:     fmt.Sprintf(format, args...),
		Fixable:     fixable,
		Instruction: in,
	}
}
