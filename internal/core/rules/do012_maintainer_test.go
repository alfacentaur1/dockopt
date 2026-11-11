package rules

import "testing"

func TestDO012(t *testing.T) {
	assertFindings(t, "DO012", ctxFor(t, "FROM alpine:3.20\nLABEL a=b\n"), 0)
	assertFixed(t, "DO012",
		"FROM alpine:3.20\nMAINTAINER Filip <filip@example.com>\n",
		"FROM alpine:3.20\nLABEL org.opencontainers.image.authors=\"Filip <filip@example.com>\"\n")
}
