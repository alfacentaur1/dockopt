package diff

import "testing"

func TestUnifiedEqual(t *testing.T) {
	if got := Unified("a", "b", []byte("x\n"), []byte("x\n"), 3); got != "" {
		t.Fatalf("expected empty diff, got %q", got)
	}
}

func TestUnifiedSingleChange(t *testing.T) {
	a := "FROM alpine\nRUN apk add git\nCMD sh\n"
	b := "FROM alpine\nRUN apk add --no-cache git\nCMD sh\n"
	want := `--- a/Dockerfile
+++ b/Dockerfile
@@ -1,3 +1,3 @@
 FROM alpine
-RUN apk add git
+RUN apk add --no-cache git
 CMD sh
`
	if got := Unified("a/Dockerfile", "b/Dockerfile", []byte(a), []byte(b), 3); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUnifiedSeparateHunks(t *testing.T) {
	a := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	b := "one\n2\n3\n4\n5\n6\n7\n8\n9\nten\n"
	want := `--- a
+++ b
@@ -1,2 +1,2 @@
-1
+one
 2
@@ -9,2 +9,2 @@
 9
-10
+ten
`
	if got := Unified("a", "b", []byte(a), []byte(b), 1); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUnifiedInsertion(t *testing.T) {
	a := "FROM golang\nCOPY . .\n"
	b := "FROM golang\nCOPY go.mod ./\nRUN go mod download\nCOPY . .\n"
	want := `--- a
+++ b
@@ -1,2 +1,4 @@
 FROM golang
+COPY go.mod ./
+RUN go mod download
 COPY . .
`
	if got := Unified("a", "b", []byte(a), []byte(b), 3); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
