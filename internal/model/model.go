// Package model holds dockopt's own representation of a Dockerfile.
//
// The parser converts buildkit's node tree into these types instead of
// using buildkit's AST directly, because that AST is awkward to mutate.
// Rules read this model and fixers change it. The renderer then writes it
// back, keeping the original raw text of instructions nobody touched.
//
// Like everything under internal/core, this package must stay pure:
// no I/O, no docker, no filesystem access (it also has to build for wasm).
package model

// Instruction is a single Dockerfile instruction, e.g. RUN, COPY or FROM.
type Instruction struct {
	// Cmd is the instruction keyword, e.g. "RUN".
	Cmd string
	// Flags are the --flags written before the arguments, e.g. "--from=builder".
	Flags []string
	// Args are the instruction's arguments.
	Args []string
	// JSON is true for the exec form (RUN ["go", "build"])
	// and false for the shell form (RUN go build).
	JSON bool

	// StartLine and EndLine are the lines in Dockerfile.Source this
	// instruction was parsed from (more than one with line continuations).
	StartLine int
	EndLine   int
	// Dirty marks an instruction changed by a fixer. The renderer
	// re-renders dirty instructions and copies the rest from Source as-is.
	Dirty bool
	// Deleted marks an instruction removed by a fixer. It stays in the
	// slice and is skipped when rendering; use Dockerfile.Live to skip it.
	Deleted bool
}

// Dockerfile is a parsed Dockerfile together with its original text.
type Dockerfile struct {
	// Instructions in file order, including deleted ones.
	Instructions []*Instruction
	// Source is the original file split into lines, so untouched
	// instructions can be written back exactly as they were.
	Source []string
}

// Live returns the instructions that haven't been deleted, in file order.
func (d *Dockerfile) Live() []*Instruction {
	var out []*Instruction
	for _, in := range d.Instructions {
		if !in.Deleted {
			out = append(out, in)
		}
	}
	return out
}
