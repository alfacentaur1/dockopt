// Package ci posts the dockopt report as a comment on the current merge
// request (GitLab) or pull request (GitHub). The comment carries a hidden
// marker so later runs update it instead of piling up new ones.
package ci

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Marker identifies dockopt's comment.
const Marker = "<!-- dockopt -->"

// Provider posts or updates the comment.
type Provider interface {
	Name() string
	Upsert(ctx context.Context, body string) error
}

// Env reads environment variables; os.Getenv in production.
type Env func(string) string

// ErrNotInMergeRequest means we're in CI but not in an MR/PR pipeline.
var ErrNotInMergeRequest = errors.New("not running for a merge request")

// Detect picks the provider from the CI environment. It returns
// (nil, nil) outside of CI, so callers can just print the comment.
func Detect(env Env, client *http.Client) (Provider, error) {
	switch {
	case env("GITHUB_ACTIONS") == "true":
		return NewGitHub(env, client)
	case env("GITLAB_CI") == "true":
		return NewGitLab(env, client)
	}
	return nil, nil
}

// ByName returns a provider by name ("github" or "gitlab").
func ByName(name string, env Env, client *http.Client) (Provider, error) {
	switch name {
	case "github":
		return NewGitHub(env, client)
	case "gitlab":
		return NewGitLab(env, client)
	}
	return nil, fmt.Errorf("unknown provider %q (use github, gitlab, auto or none)", name)
}

// comment is the part of a comment/note both APIs agree on.
type comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// api is a tiny JSON-over-HTTP helper shared by both providers.
type api struct {
	client  *http.Client
	headers map[string]string
}

func newAPI(client *http.Client, headers map[string]string) api {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return api{client: client, headers: headers}
}

func (a api) do(ctx context.Context, method, url string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("%s %s: %s: %s", method, url, resp.Status, strings.TrimSpace(string(msg)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// upsert updates the comment containing Marker or creates a new one.
func upsert(ctx context.Context, list func() ([]comment, error), update func(id int64) error, create func() error) error {
	comments, err := list()
	if err != nil {
		return err
	}
	for _, c := range comments {
		if strings.Contains(c.Body, Marker) {
			return update(c.ID)
		}
	}
	return create()
}

// WithMarker makes sure the body carries the marker.
func WithMarker(body string) string {
	if strings.Contains(body, Marker) {
		return body
	}
	return Marker + "\n" + body
}
