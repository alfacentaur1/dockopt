package rules

import "testing"

func TestDO010(t *testing.T) {
	assertFindings(t, "DO010", ctxFor(t, "FROM golang:1.22\nCOPY . .\nRUN go build -o /app\nCMD [\"/app\"]\n"), 1)
	assertFindings(t, "DO010", ctxFor(t, "FROM golang:1.22 AS build\nRUN go build -o /app\nFROM alpine:3.20\nCOPY --from=build /app /app\n"), 0)
	assertFindings(t, "DO010", ctxFor(t, "FROM golang:1.22\nRUN go test ./...\n"), 0)
}
