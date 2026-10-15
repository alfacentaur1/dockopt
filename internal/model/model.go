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

// find the last stage, needed in opt tasks
func (d *Dockerfile) finalStage() *Stage {
	stages := d.Stages()

	if len(stages) != 0 {
		return stages[len(stages)-1]
	}

	return nil
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
