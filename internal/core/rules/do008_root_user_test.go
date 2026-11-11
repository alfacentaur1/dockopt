package rules

import "testing"

func TestDO008(t *testing.T) {
	assertFindings(t, "DO008", ctxFor(t, "FROM alpine:3.20\nCMD [\"sh\"]\n"), 1)
	assertFindings(t, "DO008", ctxFor(t, "FROM alpine:3.20\nUSER app\n"), 0)
	assertFindings(t, "DO008", ctxFor(t, "FROM alpine:3.20\nUSER app\nRUN x\nUSER root\n"), 1)
	assertFindings(t, "DO008", ctxFor(t, "FROM alpine:3.20\nUSER root\nRUN x\nUSER 1000:1000\n"), 0)
	assertFindings(t, "DO008", ctxFor(t, "FROM gcr.io/distroless/static:nonroot\n"), 0)
	// only the final stage matters
	assertFindings(t, "DO008", ctxFor(t, "FROM golang:1.22 AS build\nRUN go build\nFROM alpine:3.20\nUSER 65534\n"), 0)
}
