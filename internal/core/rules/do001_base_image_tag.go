package rules

import "strings"

type baseImageTag struct{ meta }

func init() {
	Register(baseImageTag{meta{
		id:    "DO001",
		title: "Pin the base image version",
		desc: "FROM without a tag, or with :latest, pulls whatever is newest at build time. " +
			"Builds stop being reproducible and the layer cache breaks without anyone changing the Dockerfile.",
		sev: Warning,
	}})
}

func (r baseImageTag) Check(ctx *Context) []Finding {
	var out []Finding
	stages := map[string]bool{}

	for _, st := range ctx.File.Stages() {
		img := st.BaseImage
		switch {
		case strings.EqualFold(img, "scratch"),
			strings.Contains(img, "$"), // ARG driven, can't know
			strings.Contains(img, "@sha256:"),
			stages[strings.ToLower(img)]: // FROM build, an earlier stage
		default:
			switch tag := imageTag(img); tag {
			case "":
				out = append(out, r.finding(st.From, false, "%s has no tag, pin a version (e.g. %s:<version>)", img, img))
			case "latest":
				out = append(out, r.finding(st.From, false, "%s uses :latest, pin a concrete version", img))
			}
		}
		if st.Name != "" {
			stages[strings.ToLower(st.Name)] = true
		}
	}
	return out
}

// imageTag returns the tag of an image reference. A colon before the last
// slash belongs to a registry port (localhost:5000/app), not to a tag.
func imageTag(ref string) string {
	ref = strings.SplitN(ref, "@", 2)[0]
	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")
	if colon <= slash {
		return ""
	}
	return ref[colon+1:]
}
