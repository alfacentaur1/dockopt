package ci

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeAPI records requests and serves a fixed list of existing comments.
type fakeAPI struct {
	mu       sync.Mutex
	existing []comment
	calls    []string // "METHOD path"
	bodies   []string
	headers  http.Header
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	f.headers = r.Header.Clone()
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(f.existing)
		return
	}
	b, _ := io.ReadAll(r.Body)
	var p map[string]string
	json.Unmarshal(b, &p)
	f.bodies = append(f.bodies, p["body"])
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("{}"))
}

func env(m map[string]string) Env { return func(k string) string { return m[k] } }

func TestDetect(t *testing.T) {
	p, err := Detect(env(nil), nil)
	if p != nil || err != nil {
		t.Fatalf("outside CI: %v, %v", p, err)
	}
	if _, err := Detect(env(map[string]string{"GITLAB_CI": "true", "DOCKOPT_GITLAB_TOKEN": "x", "CI_PROJECT_ID": "1"}), nil); !errors.Is(err, ErrNotInMergeRequest) {
		t.Fatalf("gitlab branch pipeline should report ErrNotInMergeRequest, got %v", err)
	}
	if _, err := Detect(env(map[string]string{"GITHUB_ACTIONS": "true"}), nil); err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Fatalf("missing token should be explained, got %v", err)
	}
}

func TestGitHubCreatesComment(t *testing.T) {
	fake := &fakeAPI{existing: []comment{{ID: 1, Body: "LGTM"}}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	event := filepath.Join(t.TempDir(), "event.json")
	os.WriteFile(event, []byte(`{"pull_request":{"number":42}}`), 0o644)

	gh, err := NewGitHub(env(map[string]string{
		"GITHUB_TOKEN": "secret", "GITHUB_REPOSITORY": "me/app", "GITHUB_EVENT_PATH": event, "GITHUB_API_URL": srv.URL,
	}), srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := gh.Upsert(context.Background(), "report"); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /repos/me/app/issues/42/comments", "POST /repos/me/app/issues/42/comments"}
	if strings.Join(fake.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls %v", fake.calls)
	}
	if fake.bodies[0] != Marker+"\nreport" {
		t.Errorf("body %q", fake.bodies[0])
	}
	if fake.headers.Get("Authorization") != "Bearer secret" {
		t.Errorf("auth header %q", fake.headers.Get("Authorization"))
	}
}

func TestGitHubUpdatesItsOwnComment(t *testing.T) {
	fake := &fakeAPI{existing: []comment{{ID: 1, Body: "LGTM"}, {ID: 7, Body: Marker + "\nold"}}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	gh, _ := NewGitHub(env(map[string]string{
		"GITHUB_TOKEN": "t", "GITHUB_REPOSITORY": "me/app", "GITHUB_REF": "refs/pull/5/merge", "GITHUB_API_URL": srv.URL,
	}), srv.Client())
	if err := gh.Upsert(context.Background(), "new"); err != nil {
		t.Fatal(err)
	}
	if fake.calls[1] != "PATCH /repos/me/app/issues/comments/7" {
		t.Fatalf("calls %v", fake.calls)
	}
}

func TestGitLabUpsert(t *testing.T) {
	fake := &fakeAPI{existing: []comment{{ID: 3, Body: Marker + "\nold"}}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	gl, err := NewGitLab(env(map[string]string{
		"DOCKOPT_GITLAB_TOKEN": "glpat", "CI_MERGE_REQUEST_IID": "12", "CI_PROJECT_ID": "99", "CI_API_V4_URL": srv.URL,
	}), srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := gl.Upsert(context.Background(), "report"); err != nil {
		t.Fatal(err)
	}
	if fake.calls[1] != "PUT /projects/99/merge_requests/12/notes/3" {
		t.Fatalf("calls %v", fake.calls)
	}
	if fake.headers.Get("PRIVATE-TOKEN") != "glpat" {
		t.Errorf("token header %q", fake.headers.Get("PRIVATE-TOKEN"))
	}
}

func TestAPIErrorsAreReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()
	gl, _ := NewGitLab(env(map[string]string{
		"GITLAB_TOKEN": "x", "CI_MERGE_REQUEST_IID": "1", "CI_PROJECT_ID": "1", "CI_API_V4_URL": srv.URL,
	}), srv.Client())
	err := gl.Upsert(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "Bad credentials") {
		t.Fatalf("got %v", err)
	}
}
