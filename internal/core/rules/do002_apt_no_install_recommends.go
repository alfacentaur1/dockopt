package rules

import "regexp"

type aptNoRecommends struct{ meta }

var (
	aptInstall    = regexp.MustCompile(`\bapt-get(?:\s+-{1,2}[\w-]+(?:=\S+)?)*\s+install\b`)
	recommendsOff = regexp.MustCompile(`Install-Recommends=(false|0)`)
)

const noRecommends = "--no-install-recommends"

func init() {
	Register(aptNoRecommends{meta{
		id:    "DO002",
		title: "Skip recommended apt packages",
		desc: "apt-get install pulls in \"recommended\" packages by default. " +
			"--no-install-recommends usually shaves tens of megabytes off the image.",
		sev: Warning,
	}})
}

func (r aptNoRecommends) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd != "RUN" {
			continue
		}
		cmd := command(in)
		if missingFlag(cmd, aptInstall, noRecommends) && !recommendsOff.MatchString(cmd) {
			out = append(out, r.finding(in, !in.JSON, "apt-get install without %s", noRecommends))
		}
	}
	return out
}

func (r aptNoRecommends) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.JSON {
		return false
	}
	raw, ok := insertFlag(ctx.File.RawText(in), aptInstall, noRecommends)
	if ok {
		ctx.File.Replace(in, raw)
	}
	return ok
}
