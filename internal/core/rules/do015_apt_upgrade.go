package rules

import "regexp"

type aptUpgrade struct{ meta }

var aptUpgradeRe = regexp.MustCompile(`\bapt(?:-get)?(?:\s+-{1,2}[\w-]+(?:=\S+)?)*\s+(?:dist-|full-)?upgrade\b`)

func init() {
	Register(aptUpgrade{meta{
		id:    "DO015",
		title: "Don't upgrade packages in the image",
		desc: "apt-get upgrade makes the result depend on the day you build. " +
			"Bump the base image tag instead, it's reviewable and reproducible.",
		sev: Info,
	}})
}

func (r aptUpgrade) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd == "RUN" && aptUpgradeRe.MatchString(command(in)) {
			out = append(out, r.finding(in, false, "packages are upgraded during the build, bump the base image instead"))
		}
	}
	return out
}
