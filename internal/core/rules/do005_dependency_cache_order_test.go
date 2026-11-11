package rules

import "testing"

func TestDO005Go(t *testing.T) {
	src := "FROM golang:1.22 AS build\nWORKDIR /src\nCOPY . .\nRUN go build -o /app ./cmd/app\n"
	ctx := ctxFor(t, src)
	ctx.Lang = "go"
	ctx.Files = []string{"go.mod", "go.sum", "main.go"}

	got := fixOnce(t, "DO005", ctx)
	want := "FROM golang:1.22 AS build\nWORKDIR /src\nCOPY go.mod go.sum ./\nRUN go mod download\nCOPY . .\nRUN go build -o /app ./cmd/app\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDO005GoWithoutSumAndCustomDest(t *testing.T) {
	src := "FROM golang:1.22\nCOPY --chown=app . /src\nRUN go build ./...\n"
	ctx := ctxFor(t, src)
	ctx.Files = []string{"go.mod", "main.go"}

	got := fixOnce(t, "DO005", ctx) // language comes from the base image here
	want := "FROM golang:1.22\nCOPY --chown=app go.mod /src/\nRUN go mod download\nCOPY --chown=app . /src\nRUN go build ./...\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDO005NodeMovesInstall(t *testing.T) {
	src := "FROM node:20-alpine\nWORKDIR /app\nCOPY . .\nRUN npm ci --omit=dev\nCMD [\"node\", \"index.js\"]\n"
	ctx := ctxFor(t, src)
	ctx.Lang = "node"
	ctx.Files = []string{"package.json", "package-lock.json", "index.js"}

	got := fixOnce(t, "DO005", ctx)
	want := "FROM node:20-alpine\nWORKDIR /app\nCOPY package*.json ./\nRUN npm ci --omit=dev\nCOPY . .\nCMD [\"node\", \"index.js\"]\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDO005NodeNotMovableWhenInstallDoesMore(t *testing.T) {
	ctx := ctxFor(t, "FROM node:20\nCOPY . .\nRUN npm ci && npm run build\n")
	f := assertFindings(t, "DO005", ctx, 1)
	if f[0].Fixable {
		t.Fatal("an install chained with a build can't be moved safely")
	}
}

func TestDO005AlreadyOrdered(t *testing.T) {
	ctx := ctxFor(t, "FROM python:3.12\nCOPY requirements.txt .\nRUN pip install -r requirements.txt\nCOPY . .\n")
	assertFindings(t, "DO005", ctx, 0)
}

func TestDO005Python(t *testing.T) {
	src := "FROM python:3.12-slim\nCOPY . .\nRUN pip install --no-cache-dir -r requirements.txt\n"
	got := fixOnce(t, "DO005", ctxFor(t, src))
	want := "FROM python:3.12-slim\nCOPY requirements.txt ./\nRUN pip install --no-cache-dir -r requirements.txt\nCOPY . .\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDO005UnknownLanguage(t *testing.T) {
	assertFindings(t, "DO005", ctxFor(t, "FROM alpine:3.20\nCOPY . .\nRUN make\n"), 0)
}
