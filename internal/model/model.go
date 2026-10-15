// Package model is our own Dockerfile representation. buildkit's AST is
// awkward to mutate, so the parser converts it into this.
package model

type Instruction struct {
	Cmd   string
	Flags []string
	Args  []string
	JSON  bool // exec form: RUN ["a", "b"]

	StartLine int
	EndLine   int
	Dirty     bool // changed by a fixer, needs re-rendering
	Deleted   bool // removed by a fixer, kept so line numbers stay valid
}

type Dockerfile struct {
	Instructions []*Instruction
	Source       []string // original lines, untouched instructions are copied from here
}

// Live returns instructions that aren't deleted.
func (d *Dockerfile) Live() []*Instruction {
	var out []*Instruction
	for _, in := range d.Instructions {
		if !in.Deleted {
			out = append(out, in)
		}
	}
	return out
}
