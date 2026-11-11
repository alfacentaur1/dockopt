package model

import "testing"

func TestLiveSkipsDeleted(t *testing.T) {
	df := &Dockerfile{Instructions: []*Instruction{
		{Cmd: "FROM"},
		{Cmd: "RUN", Deleted: true},
		{Cmd: "CMD"},
	}}

	got := df.Live()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}

func TestRawTextJoinsContinuationLines(t *testing.T) {
	df := &Dockerfile{Source: []string{"FROM alpine", "RUN a \\", "    && b"}}
	in := &Instruction{Cmd: "RUN", StartLine: 2, EndLine: 3}

	if got, want := df.RawText(in), "RUN a \\\n    && b"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRawTextOfNewInstructionIsRaw(t *testing.T) {
	df := &Dockerfile{}
	in := &Instruction{Cmd: "RUN", Raw: "RUN true"}

	if got := df.RawText(in); got != "RUN true" {
		t.Fatalf("got %q", got)
	}
}

func TestInsertBefore(t *testing.T) {
	a, b, c := &Instruction{Cmd: "FROM"}, &Instruction{Cmd: "COPY"}, &Instruction{Cmd: "RUN"}
	df := &Dockerfile{Instructions: []*Instruction{a, b}}

	df.InsertBefore(b, c)
	if df.Instructions[1] != c || df.Instructions[2] != b {
		t.Fatalf("unexpected order: %v", df.Instructions)
	}
}

func TestFinalStage(t *testing.T) {
	df := &Dockerfile{Instructions: []*Instruction{
		{Cmd: "FROM", Args: []string{"golang", "AS", "build"}},
		{Cmd: "RUN", Args: []string{"go build"}},
		{Cmd: "FROM", Args: []string{"alpine"}},
		{Cmd: "CMD", Args: []string{"/app"}},
	}}

	final := df.FinalStage()
	if final == nil || final.BaseImage != "alpine" || len(final.Instructions) != 1 {
		t.Fatalf("unexpected final stage: %+v", final)
	}
	if df.Stages()[0].Name != "build" {
		t.Fatalf("first stage should be named build")
	}
}
