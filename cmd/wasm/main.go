//go:build js && wasm

// Command wasm exposes the dockopt core to JavaScript for the web demo.
//
//	dockopt.analyze(dockerfile, options) -> JSON string
//	dockopt.fix(dockerfile, options)     -> JSON string
//	dockopt.rules()                      -> JSON string
//
// options: { dockerignore: string | null, files: string[], ignore: string[] }
//
// Everything runs in the browser; nothing is sent anywhere.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/alfacentaur1/dockopt/internal/core"
	"github.com/alfacentaur1/dockopt/internal/core/rules"
	"github.com/alfacentaur1/dockopt/internal/diff"
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("analyze", js.FuncOf(analyze))
	api.Set("fix", js.FuncOf(fix))
	api.Set("rules", js.FuncOf(listRules))
	js.Global().Set("dockopt", api)
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("dockopt-ready"))
	select {} // keep the Go runtime alive for callbacks
}

func input(args []js.Value) (core.Input, core.Options) {
	var in core.Input
	var opts core.Options
	if len(args) > 0 {
		in.Dockerfile = []byte(args[0].String())
	}
	if len(args) > 1 && args[1].Type() == js.TypeObject {
		o := args[1]
		if v := o.Get("dockerignore"); v.Type() == js.TypeString {
			in.DockerIgnore = []byte(v.String())
		}
		in.Files = strings(o.Get("files"))
		opts.Ignore = strings(o.Get("ignore"))
	}
	return in, opts
}

func strings(v js.Value) []string {
	if v.Type() != js.TypeObject || !v.InstanceOf(js.Global().Get("Array")) {
		return nil
	}
	out := make([]string, v.Length())
	for i := range out {
		out[i] = v.Index(i).String()
	}
	return out
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}
	return string(b)
}

type errResult struct {
	Error string `json:"error"`
}

func analyze(_ js.Value, args []js.Value) any {
	in, opts := input(args)
	res, err := core.Analyze(in, opts)
	if err != nil {
		return toJSON(errResult{err.Error()})
	}
	return toJSON(res)
}

func fix(_ js.Value, args []js.Value) any {
	in, opts := input(args)
	res, err := core.Fix(in, opts)
	if err != nil {
		return toJSON(errResult{err.Error()})
	}
	return toJSON(struct {
		*core.FixResult
		Output string `json:"output"`
		Diff   string `json:"diff"`
	}{res, string(res.Output), diff.Unified("a/Dockerfile", "b/Dockerfile", in.Dockerfile, res.Output, 3)})
}

func listRules(js.Value, []js.Value) any {
	type row struct {
		ID          string         `json:"id"`
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Severity    rules.Severity `json:"severity"`
		Fixable     bool           `json:"fixable"`
	}
	var out []row
	for _, r := range rules.All() {
		out = append(out, row{r.ID(), r.Title(), r.Description(), r.Severity(), rules.IsFixable(r)})
	}
	return toJSON(out)
}
