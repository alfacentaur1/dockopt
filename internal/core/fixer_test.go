package core

import (
	"testing"
)

func TestFixEndToEnd(t *testing.T) {
	src := `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build
WORKDIR /src
# toolchain deps
RUN apt-get update
RUN apt-get install -y git
COPY . .
RUN go build -o /out/app ./cmd/app

FROM alpine:3.20
RUN apk add ca-certificates
COPY --from=build /out/app /app
USER 65534
ENTRYPOINT /app serve
`
	want := `# syntax=docker/dockerfile:1
FROM golang:1.22 AS build
WORKDIR /src
# toolchain deps
RUN apt-get update \
    && apt-get install --no-install-recommends -y git && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/app ./cmd/app

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/app /app
USER 65534
ENTRYPOINT ["/app", "serve"]
`
	res, err := Fix(Input{Dockerfile: []byte(src), DockerIgnore: []byte(".git\n"), Files: []string{"go.mod", "go.sum"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(res.Output); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s\napplied: %+v", got, want, res.Applied)
	}
	if len(res.Remaining) != 0 {
		t.Errorf("nothing should remain, got %+v", res.Remaining)
	}
}

func TestFixIsIdempotent(t *testing.T) {
	src := "FROM debian:12\nRUN apt-get update && apt-get install -y curl\nUSER app\n"
	first, err := Fix(Input{Dockerfile: []byte(src)}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Fix(Input{Dockerfile: first.Output}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed() {
		t.Fatalf("second run changed the file again: %+v", second.Applied)
	}
}

func TestFixLeavesCleanFilesAlone(t *testing.T) {
	src := "FROM alpine:3.20\nUSER app\nCMD [\"sh\"]" // note: no trailing newline
	res, err := Fix(Input{Dockerfile: []byte(src)}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed() || string(res.Output) != src {
		t.Fatalf("clean file was modified: %q", res.Output)
	}
}

func TestFixRespectsIgnore(t *testing.T) {
	src := "FROM alpine:3.20\nRUN apk add git\nUSER app\n"
	res, _ := Fix(Input{Dockerfile: []byte(src)}, Options{Ignore: []string{"DO013"}})
	if res.Changed() {
		t.Fatalf("DO013 was ignored, nothing to fix: %+v", res.Applied)
	}
}
