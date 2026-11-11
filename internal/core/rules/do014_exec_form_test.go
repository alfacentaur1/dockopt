package rules

import "testing"

func TestDO014Check(t *testing.T) {
	assertFindings(t, "DO014", ctxFor(t, "FROM alpine:3.20\nCMD [\"/app\"]\n"), 0)
	f := assertFindings(t, "DO014", ctxFor(t, "FROM alpine:3.20\nCMD /app --port $PORT\n"), 1)
	if f[0].Fixable {
		t.Error("variables need a shell, can't convert")
	}
}

func TestDO014Fix(t *testing.T) {
	assertFixed(t, "DO014",
		"FROM alpine:3.20\nENTRYPOINT /usr/local/bin/app serve --port 8080\n",
		"FROM alpine:3.20\nENTRYPOINT [\"/usr/local/bin/app\", \"serve\", \"--port\", \"8080\"]\n")
}
