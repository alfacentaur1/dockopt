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
