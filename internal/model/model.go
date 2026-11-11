// Package model is our own Dockerfile representation. buildkit's AST is
// awkward to mutate, so the parser converts it into this.
package model

import (
	"slices"
	"strings"
)

type Instruction struct {
	Cmd   string
	Flags []string
	Args  []string
	JSON  bool // exec form: RUN ["a", "b"]

	StartLine int // 0 for instructions added by a fixer
	EndLine   int
	Dirty     bool   // changed by a fixer, needs re-rendering
	Deleted   bool   // removed by a fixer, kept so line numbers stay valid
	Raw       string // replacement text; when set, the renderer writes it verbatim
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

// every stage beggins with from instruction
// extract from block from dockerfile
type Stage struct {
	Index        int
	Name         string //nullable
	BaseImage    string
	From         *Instruction //nullable
	Instructions []*Instruction
}

func (d *Dockerfile) Stages() []*Stage {
	var stages []*Stage
	var cur *Stage

	for _, in := range d.Live() {
		if in.Cmd == "FROM" {
			cur = &Stage{Index: len(stages), From: in, BaseImage: in.Args[0]}
			if (len(in.Args) >= 3) && strings.EqualFold(in.Args[1], "AS") {
				cur.Name = in.Args[2]
			}
			stages = append(stages, cur)
			continue

		}

		if cur == nil {
			continue
		}

		//append instructions from dockerfile
		cur.Instructions = append(cur.Instructions, in)
	}

	return stages

}

// FinalStage is the stage that ends up in the image, i.e. the last one.
func (d *Dockerfile) FinalStage() *Stage {
	stages := d.Stages()

	if len(stages) != 0 {
		return stages[len(stages)-1]
	}

	return nil
}

// RawText returns the original source text of an instruction, continuation
// lines included. Instructions added by a fixer have no source and return Raw.
func (d *Dockerfile) RawText(in *Instruction) string {
	if in.StartLine <= 0 || in.EndLine > len(d.Source) {
		return in.Raw
	}
	return strings.Join(d.Source[in.StartLine-1:in.EndLine], "\n")
}

// Replace marks in as changed and gives it new text.
func (d *Dockerfile) Replace(in *Instruction, raw string) {
	in.Raw = raw
	in.Dirty = true
}

// Delete marks in as removed.
func (d *Dockerfile) Delete(in *Instruction) {
	in.Deleted = true
}

// InsertBefore puts in right before target, if target isn't in the list,
// in is appended at the end
func (d *Dockerfile) InsertBefore(target, in *Instruction) {
	for i, existing := range d.Instructions {
		if existing == target {
			d.Instructions = slices.Insert(d.Instructions, i, in)
			return
		}
	}
	d.Instructions = append(d.Instructions, in)
}

// InsertAfter puts in right after target, if target isn't in the list,
// in is appended at the end
func (d *Dockerfile) InsertAfter(target, in *Instruction) {
	for i, existing := range d.Instructions {
		if existing == target {
			d.Instructions = slices.Insert(d.Instructions, i+1, in)
			return
		}
	}
	d.Instructions = append(d.Instructions, in)
}
