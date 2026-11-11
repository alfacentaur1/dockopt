package rules

import (
	"regexp"
	"strings"
)

type pipNoCache struct{ meta }

var pipInstall = regexp.MustCompile(`\b(?:pip3?|python3? -m pip)(?:\s+-{1,2}[\w-]+(?:=\S+)?)*\s+install\b`)

const noCacheDir = "--no-cache-dir"

func init() {
	Register(pipNoCache{meta{
		id:    "DO009",
		title: "Don't keep the pip cache",
		desc: "pip keeps every downloaded wheel in ~/.cache/pip. In an image that's dead weight; " +
			"use --no-cache-dir, PIP_NO_CACHE_DIR=1 or a cache mount.",
		sev: Info,
	}})
}

func (r pipNoCache) Check(ctx *Context) []Finding {
	for _, in := range ctx.File.Live() {
		if in.Cmd == "ENV" && strings.Contains(strings.Join(in.Args, " "), "PIP_NO_CACHE_DIR") {
			return nil
		}
	}
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd != "RUN" || hasCacheMount(in, ".cache/pip") {
			continue
		}
		if missingFlag(command(in), pipInstall, noCacheDir) {
			out = append(out, r.finding(in, !in.JSON, "pip install without %s", noCacheDir))
		}
	}
	return out
}

func (r pipNoCache) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.JSON {
		return false
	}
	raw, ok := insertFlag(ctx.File.RawText(in), pipInstall, noCacheDir)
	if ok {
		ctx.File.Replace(in, raw)
	}
	return ok
}
