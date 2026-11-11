package rules

import "testing"

func TestDO011(t *testing.T) {
	assertFindings(t, "DO011", ctxFor(t, "FROM alpine:3.20\nRUN cd /src && make\n"), 1)
	assertFindings(t, "DO011", ctxFor(t, "FROM alpine:3.20\nRUN make && cd /tmp && ls\n"), 0)
	assertFindings(t, "DO011", ctxFor(t, "FROM alpine:3.20\nWORKDIR /src\nRUN make\n"), 0)
}
