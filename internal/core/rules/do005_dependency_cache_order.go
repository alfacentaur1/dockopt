package rules

import (
	"regexp"
	"strings"

	"github.com/alfacentaur1/dockopt/internal/model"
)

type depCacheOrder struct{ meta }

func init() {
	Register(depCacheOrder{meta{
		id:    "DO005",
		title: "Install dependencies before copying the source",
		desc: "COPY . . followed by a dependency install means any source change invalidates the install layer, " +
			"so every build downloads all dependencies again. Copy the manifest (go.mod, package.json, " +
			"requirements.txt) first, install, then copy the rest.",
		sev: Warning,
	}})
}

// ecosystem describes how one language installs dependencies.
type ecosystem struct {
	name string
	// manifests to copy before the source, given what's in the build context
	manifests func(ctx *Context) []string
	// a RUN that needs the dependencies (download, build, install...)
	uses *regexp.Regexp
	// a RUN that does nothing but install dependencies; only those can be moved
	pureInstall *regexp.Regexp
	// command to insert when there is nothing to move (Go only)
	download string
}

var ecosystems = map[string]ecosystem{
	"go": {
		name: "Go modules",
		manifests: func(ctx *Context) []string {
			if ctx.HasFile("go.sum") || len(ctx.Files) == 0 {
				return []string{"go.mod", "go.sum"}
			}
			return []string{"go.mod"}
		},
		uses:     regexp.MustCompile(`\bgo (build|mod download|test|install|run|generate)\b`),
		download: "go mod download",
	},
	"node": {
		name: "npm packages",
		manifests: func(ctx *Context) []string {
			files := []string{"package*.json"}
			for _, lock := range []string{"yarn.lock", "pnpm-lock.yaml"} {
				if ctx.HasFile(lock) {
					files = append(files, lock)
				}
			}
			return files
		},
		uses:        regexp.MustCompile(`\b(npm (ci|install|i)|yarn( install)?|pnpm (install|i))\b`),
		pureInstall: regexp.MustCompile(`^(npm (ci|install|i)|yarn( install)?|pnpm (install|i))( [^&|;]*)?$`),
	},
	"python": {
		name:      "pip packages",
		manifests: func(*Context) []string { return []string{"requirements.txt"} },
		uses:      regexp.MustCompile(`\b(pip3?|python3? -m pip) install\b[^&|;]*-r\s*requirements\.txt`),
		pureInstall: regexp.MustCompile(
			`^(pip3?|python3? -m pip) install( [^&|;]*)? -r\s*requirements\.txt( [^&|;]*)?$`),
	},
}

// stageLang guesses the language of a stage: the project language if known,
// otherwise from the base image.
func stageLang(ctx *Context, st *model.Stage) string {
	if ctx.Lang != "" {
		return ctx.Lang
	}
	img := strings.ToLower(st.BaseImage)
	img = img[strings.LastIndex(img, "/")+1:]
	switch {
	case strings.HasPrefix(img, "golang"):
		return "go"
	case strings.HasPrefix(img, "node"):
		return "node"
	case strings.HasPrefix(img, "python"):
		return "python"
	}
	return ""
}

// copiesContext reports whether in is a COPY/ADD of the whole build context.
func copiesContext(in *model.Instruction) bool {
	if (in.Cmd != "COPY" && in.Cmd != "ADD") || len(in.Args) < 2 {
		return false
	}
	for _, f := range in.Flags {
		if strings.HasPrefix(f, "--from") {
			return false
		}
	}
	for _, src := range in.Args[:len(in.Args)-1] {
		if src == "." || src == "./" {
			return true
		}
	}
	return false
}

// copiesManifest reports whether in copies one of the manifest files.
func copiesManifest(in *model.Instruction, manifests []string) bool {
	if in.Cmd != "COPY" && in.Cmd != "ADD" || len(in.Args) < 2 {
		return false
	}
	for _, src := range in.Args[:len(in.Args)-1] {
		base := src[strings.LastIndex(src, "/")+1:]
		for _, m := range manifests {
			if base == m || strings.HasPrefix(m, "package") && strings.HasPrefix(base, "package") {
				return true
			}
		}
	}
	return false
}

// problem is what Check found and Fix needs.
type depProblem struct {
	eco      ecosystem
	copyAll  *model.Instruction
	install  *model.Instruction // first RUN that needs the dependencies
	movable  bool
	manifest []string
}

func (r depCacheOrder) find(ctx *Context) []depProblem {
	var out []depProblem
	for _, st := range ctx.File.Stages() {
		eco, ok := ecosystems[stageLang(ctx, st)]
		if !ok {
			continue
		}
		manifests := eco.manifests(ctx)
		for i, in := range st.Instructions {
			if copiesManifest(in, manifests) {
				break // dependencies are already copied first
			}
			if !copiesContext(in) {
				continue
			}
			for _, later := range st.Instructions[i+1:] {
				if later.Cmd == "RUN" && eco.uses.MatchString(command(later)) {
					movable := eco.download != "" ||
						(eco.pureInstall != nil && !later.JSON && eco.pureInstall.MatchString(command(later)))
					out = append(out, depProblem{eco: eco, copyAll: in, install: later, movable: movable, manifest: manifests})
					break
				}
			}
			break
		}
	}
	return out
}

func (r depCacheOrder) Check(ctx *Context) []Finding {
	var out []Finding
	for _, p := range r.find(ctx) {
		out = append(out, r.finding(p.copyAll, p.movable,
			"the whole context is copied before %s are installed (line %d); copy %s first so the install layer stays cached",
			p.eco.name, p.install.StartLine, strings.Join(p.manifest, " ")))
	}
	return out
}

func (r depCacheOrder) Fix(ctx *Context, f Finding) bool {
	for _, p := range r.find(ctx) {
		if p.copyAll != f.Instruction || !p.movable {
			continue
		}

		dst := p.copyAll.Args[len(p.copyAll.Args)-1]
		if dst == "." || dst == "./" {
			dst = "./"
		} else if !strings.HasSuffix(dst, "/") {
			dst += "/"
		}
		parts := append([]string{"COPY"}, p.copyAll.Flags...) // keep --chown and friends
		parts = append(parts, p.manifest...)
		parts = append(parts, dst)
		ctx.File.InsertBefore(p.copyAll, &model.Instruction{Cmd: "COPY", Raw: strings.Join(parts, " "), Dirty: true})

		if p.eco.download != "" {
			ctx.File.InsertBefore(p.copyAll, &model.Instruction{Cmd: "RUN", Raw: "RUN " + p.eco.download, Dirty: true})
		} else {
			moved := &model.Instruction{Cmd: "RUN", Raw: ctx.File.RawText(p.install), Dirty: true}
			ctx.File.Delete(p.install)
			ctx.File.InsertBefore(p.copyAll, moved)
		}
		return true
	}
	return false
}
