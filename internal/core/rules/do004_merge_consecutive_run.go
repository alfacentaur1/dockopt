package rules

import (
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

type mergeRuns struct{ meta }

func init() {
	Register(mergeRuns{meta{
		id:    "DO004",
		title: "Merge consecutive RUN instructions",
		desc: "Every RUN creates a layer. Consecutive RUNs that always change together are cheaper as one, " +
			"and files deleted in a later RUN are only really gone if they were created in the same layer.",
		sev: Info,
	}})
}

// mergeable is a plain shell-form RUN: no flags (mounts, network...) and no heredoc.
func mergeable(in *model.Instruction) bool {
	return in.Cmd == "RUN" && !in.JSON && len(in.Flags) == 0 && !strings.Contains(strings.Join(in.Args, " "), "<<")
}

func (r mergeRuns) Check(ctx *Context) []Finding {
	var out []Finding
	for _, st := range ctx.File.Stages() {
		for i := 1; i < len(st.Instructions); i++ {
			prev, cur := st.Instructions[i-1], st.Instructions[i]
			if mergeable(prev) && mergeable(cur) {
				out = append(out, r.finding(cur, true, "this RUN can be merged into the RUN on line %d", prev.StartLine))
			}
		}
	}
	return out
}

func (r mergeRuns) Fix(ctx *Context, f Finding) bool {
	cur := f.Instruction
	var prev *model.Instruction
	for _, st := range ctx.File.Stages() {
		for i := 1; i < len(st.Instructions); i++ {
			if st.Instructions[i] == cur {
				prev = st.Instructions[i-1]
			}
		}
	}
	if prev == nil || !mergeable(prev) || !mergeable(cur) {
		return false
	}

	first := strings.TrimRight(ctx.File.RawText(prev), " \t")
	second := strings.TrimSpace(ctx.File.RawText(cur))
	second = strings.TrimSpace(second[len("RUN"):]) // drop the keyword, keep its continuation lines

	indent := "    "
	if strings.Contains(first, "\n") {
		indent = continuationIndent(first)
	}
	ctx.File.Replace(prev, first+" \\\n"+indent+"&& "+second)
	ctx.File.Delete(cur)
	return true
}
