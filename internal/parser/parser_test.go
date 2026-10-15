package parser

import (
	"strings"
	"testing"
	"github.com/alfacentaur1/dockopt/internal/model"
)

const multiStage = `ARG GO_VERSION=1.22

# build stage
FROM golang:${GO_VERSION} AS build
WORKDIR /src
RUN apt-get update && \
    apt-get install -y git
COPY . .

FROM alpine:3.20
COPY --from=build /out/app /app
CMD ["/app"]
`

func mustParse(t *testing.T, src string) *model.Dockerfile{
	t.Helper()
	df, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	return df
}

func TestParseInstructionCount(t *testing.T) {
	df := mustParse(t, multiStage)

	// ARG, FROM, WORKDIR, RUN, COPY, FROM, COPY, CMD
	if got := len(df.Instructions); got != 8 {
		t.Fatalf("expected 8 instructions, got %d", got)
	}
}

func TestParseCmdIsUppercase(t *testing.T) {
	df := mustParse(t, "from alpine\nrun echo hi\n")

	for _, in := range df.Instructions {
		if in.Cmd != strings.ToUpper(in.Cmd) {
			t.Errorf("cmd %q is not uppercase", in.Cmd)
		}
	}
}

func TestParseMultilineRun(t *testing.T) {
	df := mustParse(t, multiStage)
	run := df.Instructions[3]

	if run.Cmd != "RUN" {
		t.Fatalf("expected RUN, got %s", run.Cmd)
	}
	if run.StartLine != 6 || run.EndLine != 7 {
		t.Errorf("lines = %d-%d, want 6-7", run.StartLine, run.EndLine)
	}
	// shell form: the whole command is one arg
	if len(run.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d: %v", len(run.Args), run.Args)
	}
	if !strings.Contains(run.Args[0], "apt-get update") || !strings.Contains(run.Args[0], "apt-get install -y git") {
		t.Errorf("unexpected RUN args: %q", run.Args[0])
	}
	if run.JSON {
		t.Error("shell form RUN should not be JSON")
	}
}

func TestParseFlags(t *testing.T) {
	df := mustParse(t, multiStage)
	cp := df.Instructions[6]

	if cp.Cmd != "COPY" {
		t.Fatalf("expected COPY, got %s", cp.Cmd)
	}
	if len(cp.Flags) != 1 || cp.Flags[0] != "--from=build" {
		t.Errorf("flags = %v", cp.Flags)
	}
	if len(cp.Args) != 2 || cp.Args[0] != "/out/app" || cp.Args[1] != "/app" {
		t.Errorf("args = %v", cp.Args)
	}
}

func TestParseExecForm(t *testing.T) {
	df := mustParse(t, multiStage)
	cmd := df.Instructions[7]

	if !cmd.JSON {
		t.Error("CMD [\"/app\"] should be JSON")
	}
	if len(cmd.Args) != 1 || cmd.Args[0] != "/app" {
		t.Errorf("args = %v", cmd.Args)
	}
}

func TestParseStages(t *testing.T) {
	stages := mustParse(t, multiStage).Stages()

	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}
	if stages[0].Name != "build" {
		t.Errorf("stage 0 name = %q", stages[0].Name)
	}
	if stages[1].BaseImage != "alpine:3.20" {
		t.Errorf("stage 1 base = %q", stages[1].BaseImage)
	}
}

func TestParseKeepsSource(t *testing.T) {
	df := mustParse(t, multiStage)

	if got := len(df.Source); got != 12 {
		t.Fatalf("expected 12 source lines, got %d", got)
	}
	if df.Source[2] != "# build stage" {
		t.Errorf("line 3 = %q", df.Source[2])
	}
}

func TestParseEmptyFails(t *testing.T) {
	if _, err := Parse([]byte("")); err == nil {
		t.Error("expected error for empty Dockerfile")
	}
}