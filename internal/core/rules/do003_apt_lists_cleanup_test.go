package rules

import "testing"

func TestDO003Check(t *testing.T) {
	assertFindings(t, "DO003", ctxFor(t, "FROM debian:12\nRUN apt-get update && apt-get install -y git\n"), 1)
	assertFindings(t, "DO003", ctxFor(t, "FROM debian:12\nRUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*\n"), 0)
	assertFindings(t, "DO003", ctxFor(t, "FROM debian:12\nRUN --mount=type=cache,target=/var/lib/apt apt-get update && apt-get install -y git\n"), 0)
}

func TestDO003FixSingleLine(t *testing.T) {
	assertFixed(t, "DO003",
		"FROM debian:12\nRUN apt-get update && apt-get install -y git\n",
		"FROM debian:12\nRUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*\n")
}

func TestDO003FixMultiLine(t *testing.T) {
	assertFixed(t, "DO003",
		"FROM debian:12\nRUN apt-get update \\\n    && apt-get install -y git\n",
		"FROM debian:12\nRUN apt-get update \\\n    && apt-get install -y git \\\n    && rm -rf /var/lib/apt/lists/*\n")
}
