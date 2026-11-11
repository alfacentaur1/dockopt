package rules

import "testing"

func TestDO013Check(t *testing.T) {
	assertFindings(t, "DO013", ctxFor(t, "FROM alpine:3.20\nRUN apk add git\n"), 1)
	assertFindings(t, "DO013", ctxFor(t, "FROM alpine:3.20\nRUN apk add --no-cache git\n"), 0)
	assertFindings(t, "DO013", ctxFor(t, "FROM alpine:3.20\nRUN apk add git && rm -rf /var/cache/apk/*\n"), 0)
}

func TestDO013Fix(t *testing.T) {
	assertFixed(t, "DO013",
		"FROM alpine:3.20\nRUN apk update && apk add --virtual build-deps gcc\n",
		"FROM alpine:3.20\nRUN apk update && apk add --no-cache --virtual build-deps gcc\n")
}
