package rules

import "testing"

func TestDO015(t *testing.T) {
	assertFindings(t, "DO015", ctxFor(t, "FROM debian:12\nRUN apt-get update && apt-get -y upgrade\n"), 1)
	assertFindings(t, "DO015", ctxFor(t, "FROM debian:12\nRUN apt-get dist-upgrade -y\n"), 1)
	assertFindings(t, "DO015", ctxFor(t, "FROM debian:12\nRUN apt-get install -y --only-upgrade openssl\n"), 0)
}
