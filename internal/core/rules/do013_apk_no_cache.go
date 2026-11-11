package rules

import (
	"regexp"
	"strings"
)

type apkNoCache struct{ meta }

var apkAdd = regexp.MustCompile(`\bapk(?:\s+-{1,2}[\w-]+(?:=\S+)?)*\s+add\b`)

func init() {
	Register(apkNoCache{meta{
		id:    "DO013",
		title: "apk add --no-cache",
		desc: "apk add keeps the package index in /var/cache/apk. " +
			"--no-cache fetches it on the fly and leaves nothing behind.",
		sev: Warning,
	}})
}

func (r apkNoCache) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd != "RUN" || hasCacheMount(in, "/var/cache/apk") {
			continue
		}
		cmd := command(in)
		if strings.Contains(cmd, "/var/cache/apk") {
			continue // cleaned up by hand
		}
		if missingFlag(cmd, apkAdd, "--no-cache") {
			out = append(out, r.finding(in, !in.JSON, "apk add without --no-cache"))
		}
	}
	return out
}

func (r apkNoCache) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.JSON {
		return false
	}
	raw, ok := insertFlag(ctx.File.RawText(in), apkAdd, "--no-cache")
	if ok {
		ctx.File.Replace(in, raw)
	}
	return ok
}
