package rules

import (
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

type rootUser struct{ meta }

func init() {
	Register(rootUser{meta{
		id:    "DO008",
		title: "Don't run as root",
		desc: "Without a USER instruction the container runs as root. If the process is compromised, " +
			"so is everything root can reach. Add a user and switch to it in the final stage.",
		sev: Warning,
	}})
}

func (r rootUser) Check(ctx *Context) []Finding {
	final := ctx.File.FinalStage()
	if final == nil {
		return nil
	}
	img := strings.ToLower(final.BaseImage)
	if strings.Contains(img, "nonroot") || strings.Contains(img, "$") {
		return nil // distroless :nonroot and friends, or unknown
	}

	// the last USER wins
	var last *model.Instruction
	for _, in := range final.Instructions {
		if in.Cmd == "USER" && len(in.Args) > 0 {
			last = in
		}
	}
	if last == nil {
		return []Finding{r.finding(final.From, false, "the final stage has no USER, the container runs as root")}
	}
	if name := strings.SplitN(last.Args[0], ":", 2)[0]; name == "root" || name == "0" {
		return []Finding{r.finding(last, false, "the final stage switches back to root")}
	}
	return nil
}
