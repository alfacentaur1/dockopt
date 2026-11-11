package rules

import "testing"

func TestDO007Check(t *testing.T) {
	assertFindings(t, "DO007", ctxFor(t, "FROM alpine:3.20\nADD app.conf /etc/app.conf\n"), 1)
	assertFindings(t, "DO007", ctxFor(t, "FROM alpine:3.20\nADD https://example.com/x /x\n"), 0)
	assertFindings(t, "DO007", ctxFor(t, "FROM alpine:3.20\nADD rootfs.tar.gz /\n"), 0)
	assertFindings(t, "DO007", ctxFor(t, "FROM alpine:3.20\nADD --checksum=sha256:abc x /x\n"), 0)
}

func TestDO007Fix(t *testing.T) {
	assertFixed(t, "DO007",
		"FROM alpine:3.20\nadd --chown=app . /app\n",
		"FROM alpine:3.20\nCOPY --chown=app . /app\n")
}
