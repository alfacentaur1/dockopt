package core

import (
	"regexp"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/model"
)

// Inline suppressions:
//
//	# dockopt:ignore DO004 DO011    the next instruction, listed rules
//	# dockopt:ignore                the next instruction, every rule
//	# dockopt:ignore-file DO008     the whole file
var ignoreComment = regexp.MustCompile(`(?i)^\s*#\s*dockopt:\s*ignore(-file)?\b(.*)$`)
var ruleID = regexp.MustCompile(`(?i)\bDO\d{3}\b`)

type suppressions struct {
	file  map[string]bool // "*" means every rule
	lines map[int]map[string]bool
}

func findSuppressions(df *model.Dockerfile) suppressions {
	s := suppressions{file: map[string]bool{}, lines: map[int]map[string]bool{}}

	for i, line := range df.Source {
		m := ignoreComment.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		ids := map[string]bool{}
		for _, id := range ruleID.FindAllString(m[2], -1) {
			ids[strings.ToUpper(id)] = true
		}
		if len(ids) == 0 {
			ids["*"] = true
		}

		if m[1] != "" {
			for id := range ids {
				s.file[id] = true
			}
			continue
		}
		// applies to the first instruction after the comment
		lineNo := i + 1
		for _, in := range df.Instructions {
			if in.StartLine > lineNo {
				if s.lines[in.StartLine] == nil {
					s.lines[in.StartLine] = map[string]bool{}
				}
				for id := range ids {
					s.lines[in.StartLine][id] = true
				}
				break
			}
		}
	}
	return s
}

func (s suppressions) allows(f rules.Finding) bool {
	if s.file["*"] || s.file[f.RuleID] {
		return false
	}
	l := s.lines[f.Line]
	return !(l["*"] || l[f.RuleID])
}
