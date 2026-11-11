package rules

import "strings"

type addInsteadOfCopy struct{ meta }

func init() {
	Register(addInsteadOfCopy{meta{
		id:    "DO007",
		title: "Prefer COPY over ADD",
		desc: "ADD also downloads URLs and unpacks archives, which makes it surprising for plain local files. " +
			"COPY does exactly one thing.",
		sev: Info,
	}})
}

var archiveExt = []string{".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz", ".tar.zst"}

// addNeedsAdd reports whether an ADD does something COPY can't.
func addNeedsAdd(args, flags []string) bool {
	for _, f := range flags {
		if strings.HasPrefix(f, "--checksum") || strings.HasPrefix(f, "--keep-git-dir") || strings.HasPrefix(f, "--unpack") {
			return true
		}
	}
	if len(args) < 2 {
		return true
	}
	for _, src := range args[:len(args)-1] {
		low := strings.ToLower(src)
		if strings.Contains(low, "://") || strings.HasPrefix(low, "git@") {
			return true
		}
		for _, ext := range archiveExt {
			if strings.HasSuffix(low, ext) {
				return true
			}
		}
	}
	return false
}

func (r addInsteadOfCopy) Check(ctx *Context) []Finding {
	var out []Finding
	for _, in := range ctx.File.Live() {
		if in.Cmd == "ADD" && !addNeedsAdd(in.Args, in.Flags) {
			out = append(out, r.finding(in, true, "ADD of local files, use COPY"))
		}
	}
	return out
}

func (r addInsteadOfCopy) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.Cmd != "ADD" {
		return false
	}
	ctx.File.Replace(in, replaceKeyword(ctx.File.RawText(in), "COPY"))
	return true
}
