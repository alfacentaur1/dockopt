package ci

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GitLab comments on merge requests through the REST API.
//
// The job token can't write notes, so it needs a project or personal access
// token with the api scope in DOCKOPT_GITLAB_TOKEN (or GITLAB_TOKEN). The
// CI_* variables are set by GitLab in merge request pipelines.
type GitLab struct {
	api     api
	base    string
	project string
	iid     string
}

// NewGitLab reads the GitLab CI environment.
func NewGitLab(env Env, client *http.Client) (*GitLab, error) {
	token := env("DOCKOPT_GITLAB_TOKEN")
	if token == "" {
		token = env("GITLAB_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("DOCKOPT_GITLAB_TOKEN is not set (a token with the api scope, as a masked CI variable)")
	}
	iid := env("CI_MERGE_REQUEST_IID")
	if iid == "" {
		return nil, ErrNotInMergeRequest
	}
	base := strings.TrimRight(env("CI_API_V4_URL"), "/")
	if base == "" {
		base = "https://gitlab.com/api/v4"
	}
	project := env("CI_PROJECT_ID")
	if project == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is not set")
	}
	return &GitLab{
		api:     newAPI(client, map[string]string{"PRIVATE-TOKEN": token}),
		base:    base,
		project: url.PathEscape(project),
		iid:     iid,
	}, nil
}

func (g *GitLab) Name() string { return "gitlab" }

func (g *GitLab) Upsert(ctx context.Context, body string) error {
	notes := fmt.Sprintf("%s/projects/%s/merge_requests/%s/notes", g.base, g.project, g.iid)
	payload := map[string]string{"body": WithMarker(body)}
	return upsert(ctx,
		func() ([]comment, error) {
			var cs []comment
			err := g.api.do(ctx, http.MethodGet, notes+"?per_page=100&sort=desc", nil, &cs)
			return cs, err
		},
		func(id int64) error {
			return g.api.do(ctx, http.MethodPut, fmt.Sprintf("%s/%d", notes, id), payload, nil)
		},
		func() error { return g.api.do(ctx, http.MethodPost, notes, payload, nil) },
	)
}
