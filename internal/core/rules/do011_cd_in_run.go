package rules

import "strings"

type cdInRun struct{ meta }

func init() {
	Register(cdInRun{meta{
		id:    "DO011",
		title: "Use WORKDIR instead of cd",
		desc: "`RUN cd /dir && ...` only changes the directory for that one RUN, which is easy to forget. " +
			"WORKDIR is explicit and applies to every following instruction.",
		sev: Info,
	}})
}

func (r cdInRun) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd != "RUN" || in.JSON {
			continue
		}
		cmds := commands(command(in))
		if len(cmds) > 1 && strings.HasPrefix(cmds[0], "cd ") {
			out = append(out, r.finding(in, false, "RUN starts with `%s`, use WORKDIR", cmds[0]))
		}
	}
	return out
}
