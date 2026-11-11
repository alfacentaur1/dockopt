package render

import (
	"testing"

	"github.com/alfacentaur1/dockopt/internal/model"
	"github.com/alfacentaur1/dockopt/internal/parser"
)

const src = `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build

# deps first
WORKDIR /src
RUN apt-get update && \
    apt-get install -y git

COPY . .
CMD ["/app"]
`

func mustParse(t *testing.T, s string) *model.Dockerfile {
	t.Helper()
	df, err := parser.Parse([]byte(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return df
}

func TestRenderUntouchedIsIdentical(t *testing.T) {
	df := mustParse(t, src)
	if got := string(Render(df)); got != src {
		t.Fatalf("round trip changed the file:\n%s", got)
	}
}

func TestRenderReplacesOnlyDirty(t *testing.T) {
	df := mustParse(t, src)
	df.Replace(df.Instructions[3], "COPY --chown=app . .")

	want := `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build

# deps first
WORKDIR /src
RUN apt-get update && \
    apt-get install -y git

COPY --chown=app . .
CMD ["/app"]
`
	if got := string(Render(df)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderDeletedAndInserted(t *testing.T) {
	df := mustParse(t, src)
	copyAll := df.Instructions[3]
	df.Delete(copyAll)
	df.InsertAfter(df.Instructions[1], &model.Instruction{Cmd: "COPY", Raw: "COPY go.mod ./", Dirty: true})

	want := `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build

# deps first
WORKDIR /src
COPY go.mod ./
RUN apt-get update && \
    apt-get install -y git

CMD ["/app"]
`
	if got := string(Render(df)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestInstructionSynthesized(t *testing.T) {
	in := &model.Instruction{Cmd: "CMD", JSON: true, Args: []string{"/app", "--port", "8080"}}
	if got, want := Instruction(in), `CMD ["/app", "--port", "8080"]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	in = &model.Instruction{Cmd: "COPY", Flags: []string{"--from=build"}, Args: []string{"/out", "/app"}}
	if got, want := Instruction(in), "COPY --from=build /out /app"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
