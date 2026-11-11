package rules

import "testing"

func TestDO002Check(t *testing.T) {
	assertFindings(t, "DO002", ctxFor(t, "FROM debian:12\nRUN apt-get update && apt-get install -y git\n"), 1)
	assertFindings(t, "DO002", ctxFor(t, "FROM debian:12\nRUN apt-get install -y --no-install-recommends git\n"), 0)
	assertFindings(t, "DO002", ctxFor(t, "FROM debian:12\nRUN apt-get -o APT::Install-Recommends=false install git\n"), 0)
	assertFindings(t, "DO002", ctxFor(t, "FROM debian:12\nRUN echo apt-get\n"), 0)
}

func TestDO002Fix(t *testing.T) {
	assertFixed(t, "DO002",
		"FROM debian:12\nRUN apt-get update && \\\n    apt-get -y install git curl\n",
		"FROM debian:12\nRUN apt-get update && \\\n    apt-get -y install --no-install-recommends git curl\n")
}

func TestDO002FixOnlyCommandsMissingTheFlag(t *testing.T) {
	assertFixed(t, "DO002",
		"FROM debian:12\nRUN apt-get install --no-install-recommends a && apt-get install b\n",
		"FROM debian:12\nRUN apt-get install --no-install-recommends a && apt-get install --no-install-recommends b\n")
}
