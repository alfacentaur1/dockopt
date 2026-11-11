package rules

import (
	"regexp"
	"strings"
)

type multiStageGo struct{ meta }

var goBuild = regexp.MustCompile(`\bgo build\b`)

func init() {
	Register(multiStageGo{meta{
		id:    "DO010",
		title: "Use a multi-stage build for compiled code",
		desc: "Building Go in the image you ship means shipping the whole toolchain and module cache (800 MB+). " +
			"Build in one stage and copy only the binary into a small runtime image.",
		sev: Warning,
	}})
}

func (r multiStageGo) Check(ctx *Context) []Finding {
	final := ctx.File.FinalStage()
	if final == nil {
		return nil
	}
	img := strings.ToLower(final.BaseImage)
	img = img[strings.LastIndex(img, "/")+1:]
	if !strings.HasPrefix(img, "golang") {
		return nil
	}
	for _, in := range final.Instructions {
		if in.Cmd == "RUN" && goBuild.MatchString(command(in)) {
			return []Finding{r.finding(final.From, false,
				"the final image is %s and builds the binary itself; build in a separate stage and copy the binary into distroless, alpine or scratch",
				final.BaseImage)}
		}
	}
	return nil
}
