package rules

import "strings"

type aptListsCleanup struct{ meta }

const aptLists = "/var/lib/apt/lists"

func init() {
	Register(aptListsCleanup{meta{
		id:    "DO003",
		title: "Remove apt lists in the same layer",
		desc: "apt-get update downloads package lists into /var/lib/apt/lists. " +
			"Unless they're removed in the same RUN, they stay in the image forever, even if a later layer deletes them.",
		sev: Warning,
	}})
}

func (r aptListsCleanup) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd != "RUN" {
			continue
		}
		cmd := command(in)
		if !aptInstall.MatchString(cmd) || strings.Contains(cmd, aptLists) || hasCacheMount(in, "/var/lib/apt") {
			continue
		}
		out = append(out, r.finding(in, !in.JSON, "apt lists are not removed in this RUN, add `rm -rf %s/*`", aptLists))
	}
	return out
}

func (r aptListsCleanup) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.JSON {
		return false
	}
	ctx.File.Replace(in, appendCommand(ctx.File.RawText(in), "rm -rf "+aptLists+"/*"))
	return true
}
