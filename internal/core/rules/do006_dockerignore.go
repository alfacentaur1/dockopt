package rules

import "strings"

type dockerignore struct{ meta }

func init() {
	Register(dockerignore{meta{
		id:    "DO006",
		title: "Use a .dockerignore",
		desc: "COPY . sends the whole build context to the image. Without a .dockerignore that includes .git, " +
			"node_modules, local build output and secrets, and any change to them busts the cache.",
		sev: Warning,
	}})
}

func (r dockerignore) Check(ctx *Context) []Finding {
	for _, in := range ctx.File.Live() {
		if !copiesContext(in) {
			continue
		}
		if !ctx.HasDockerIgnore {
			return []Finding{r.finding(in, false, "the whole context is copied but there is no .dockerignore")}
		}
		if !ignoresGit(ctx.DockerIgnore) {
			f := r.finding(in, false, ".dockerignore doesn't exclude .git, so every commit busts the cache of this COPY")
			f.Severity = Info
			return []Finding{f}
		}
		return nil
	}
	return nil
}

func ignoresGit(patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(p), "/"), "/")
		switch p {
		case ".git", "**/.git", ".*", "*", "**":
			return true
		}
	}
	return false
}
