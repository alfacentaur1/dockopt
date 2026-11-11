package rules

import "strings"

type maintainer struct{ meta }

func init() {
	Register(maintainer{meta{
		id:    "DO012",
		title: "MAINTAINER is deprecated",
		desc: "MAINTAINER has been deprecated for years. " +
			"Use the standard org.opencontainers.image.authors label instead.",
		sev: Info,
	}})
}

func (r maintainer) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd == "MAINTAINER" {
			out = append(out, r.finding(in, true, "MAINTAINER is deprecated, use LABEL org.opencontainers.image.authors"))
		}
	}
	return out
}

func (r maintainer) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.Cmd != "MAINTAINER" {
		return false
	}
	who := strings.Trim(strings.Join(in.Args, " "), `"`)
	who = strings.ReplaceAll(who, `"`, `\"`)
	ctx.File.Replace(in, `LABEL org.opencontainers.image.authors="`+who+`"`)
	return true
}
