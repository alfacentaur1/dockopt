package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// GitHub comments on pull requests through the REST API.
//
// Needs GITHUB_TOKEN with pull-requests: write. GITHUB_REPOSITORY,
// GITHUB_EVENT_PATH and GITHUB_API_URL are set by Actions.
type GitHub struct {
	api    api
	base   string
	repo   string
	number int
}

// NewGitHub reads the Actions environment.
func NewGitHub(env Env, client *http.Client) (*GitHub, error) {
	token := env("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not set (add `env: GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`)")
	}
	repo := env("GITHUB_REPOSITORY")
	if repo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}
	number, err := prNumber(env)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(env("GITHUB_API_URL"), "/")
	if base == "" {
		base = "https://api.github.com"
	}
	return &GitHub{
		api: newAPI(client, map[string]string{
			"Authorization":        "Bearer " + token,
			"Accept":               "application/vnd.github+json",
			"X-GitHub-Api-Version": "2022-11-28",
		}),
		base:   base,
		repo:   repo,
		number: number,
	}, nil
}

// prNumber comes from the event payload, or from refs/pull/<n>/merge.
func prNumber(env Env) (int, error) {
	if path := env("GITHUB_EVENT_PATH"); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			var ev struct {
				PullRequest struct {
					Number int `json:"number"`
				} `json:"pull_request"`
			}
			if json.Unmarshal(b, &ev) == nil && ev.PullRequest.Number > 0 {
				return ev.PullRequest.Number, nil
			}
		}
	}
	if ref := env("GITHUB_REF"); strings.HasPrefix(ref, "refs/pull/") {
		if n, err := strconv.Atoi(strings.Split(ref, "/")[2]); err == nil {
			return n, nil
		}
	}
	return 0, ErrNotInMergeRequest
}

func (g *GitHub) Name() string { return "github" }

func (g *GitHub) Upsert(ctx context.Context, body string) error {
	issue := fmt.Sprintf("%s/repos/%s/issues/%d/comments", g.base, g.repo, g.number)
	payload := map[string]string{"body": WithMarker(body)}
	return upsert(ctx,
		func() ([]comment, error) {
			var cs []comment
			err := g.api.do(ctx, http.MethodGet, issue+"?per_page=100", nil, &cs)
			return cs, err
		},
		func(id int64) error {
			return g.api.do(ctx, http.MethodPatch, fmt.Sprintf("%s/repos/%s/issues/comments/%d", g.base, g.repo, id), payload, nil)
		},
		func() error { return g.api.do(ctx, http.MethodPost, issue, payload, nil) },
	)
}
