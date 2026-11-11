package rules

import "testing"

func TestDO006(t *testing.T) {
	src := "FROM alpine:3.20\nCOPY . /app\n"

	ctx := ctxFor(t, src)
	assertFindings(t, "DO006", ctx, 1)

	ctx = ctxFor(t, src)
	ctx.HasDockerIgnore = true
	ctx.DockerIgnore = []string{"node_modules"}
	f := assertFindings(t, "DO006", ctx, 1)
	if f[0].Severity != Info {
		t.Errorf("missing .git entry should only be info, got %v", f[0].Severity)
	}

	ctx = ctxFor(t, src)
	ctx.HasDockerIgnore = true
	ctx.DockerIgnore = []string{"/.git/", "node_modules"}
	assertFindings(t, "DO006", ctx, 0)

	// copying single files doesn't need a .dockerignore
	assertFindings(t, "DO006", ctxFor(t, "FROM alpine:3.20\nCOPY app /app\n"), 0)
}
