package rules

import (
	"encoding/json"
	"strings"
)

type execFormCmd struct{ meta }

func init() {
	Register(execFormCmd{meta{
		id:    "DO014",
		title: "Use exec form for CMD and ENTRYPOINT",
		desc: "Shell form runs the process under /bin/sh -c, so it doesn't get SIGTERM on `docker stop` " +
			"and is killed after the timeout instead of shutting down cleanly.",
		sev: Warning,
	}})
}

// shellMeta are characters that need a real shell; commands with them can't
// be converted to exec form mechanically.
const shellMeta = "$|&;<>()*?`'\"\\\n~"

func (r execFormCmd) Check(ctx *Context) []Finding {
	final := ctx.File.FinalStage()
	if final == nil {
		return nil
	}
	var out []Finding
	for _, in := range final.Instructions {
		if (in.Cmd == "CMD" || in.Cmd == "ENTRYPOINT") && !in.JSON && len(in.Args) > 0 {
			convertible := !strings.ContainsAny(strings.Join(in.Args, " "), shellMeta)
			out = append(out, r.finding(in, convertible, "%s in shell form won't receive signals, use exec form [\"...\"]", in.Cmd))
		}
	}
	return out
}

func (r execFormCmd) Fix(ctx *Context, f Finding) bool {
	in := f.Instruction
	if in == nil || in.JSON || strings.ContainsAny(strings.Join(in.Args, " "), shellMeta) {
		return false
	}
	fields := strings.Fields(strings.Join(in.Args, " "))
	quoted := make([]string, len(fields))
	for i, a := range fields {
		b, _ := json.Marshal(a)
		quoted[i] = string(b)
	}
	ctx.File.Replace(in, in.Cmd+" ["+strings.Join(quoted, ", ")+"]")
	return true
}
