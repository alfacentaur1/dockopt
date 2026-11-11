package rules

import "testing"

func TestDO004Check(t *testing.T) {
	assertFindings(t, "DO004", ctxFor(t, "FROM alpine:3.20\nRUN a\nRUN b\nRUN c\n"), 2)
	assertFindings(t, "DO004", ctxFor(t, "FROM alpine:3.20\nRUN a\nCOPY . .\nRUN b\n"), 0)
	assertFindings(t, "DO004", ctxFor(t, "FROM alpine:3.20\nRUN a\nRUN --mount=type=cache,target=/root/.cache b\n"), 0)
	assertFindings(t, "DO004", ctxFor(t, "FROM alpine:3.20\nRUN a\nRUN [\"b\"]\n"), 0)
	// RUNs in different stages are never merged
	assertFindings(t, "DO004", ctxFor(t, "FROM alpine:3.20 AS one\nRUN a\nFROM alpine:3.20\nRUN b\n"), 0)
}

func TestDO004Fix(t *testing.T) {
	assertFixed(t, "DO004",
		"FROM alpine:3.20\nRUN apk add git\nRUN git --version\nCMD [\"sh\"]\n",
		"FROM alpine:3.20\nRUN apk add git \\\n    && git --version\nCMD [\"sh\"]\n")
}

func TestDO004FixKeepsContinuationLines(t *testing.T) {
	assertFixed(t, "DO004",
		"FROM alpine:3.20\nRUN a \\\n  && b\nRUN c \\\n  && d\n",
		"FROM alpine:3.20\nRUN a \\\n  && b \\\n  && c \\\n  && d\n")
}
