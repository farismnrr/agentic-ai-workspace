package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

const (
	pageSize     = 100
	maxRepos     = 300
	maxInstalls  = 10
	repoTimeSlot = "2006-01-02T15:04:05Z"
)

// Repos lists the repositories of every installation of the app that the person can reach, newest first. A
// GitHub App's user token cannot list "all repos"; it sees them through the installations the person made.
func (c *Client) Repos(ctx context.Context, access string) ([]domain.Repo, error) {
	var installs struct {
		Installations []struct {
			ID int64 `json:"id"`
		} `json:"installations"`
	}

	if err := c.get(ctx, access, "/user/installations?per_page="+strconv.Itoa(maxInstalls), &installs); err != nil {
		return nil, err
	}

	repos := []domain.Repo{}

	for _, install := range installs.Installations {
		var page struct {
			Repositories []struct {
				FullName    string `json:"full_name"`
				Description string `json:"description"`
				PushedAt    string `json:"pushed_at"`
				Private     bool   `json:"private"`
			} `json:"repositories"`
		}

		path := "/user/installations/" + strconv.FormatInt(install.ID, 10) + "/repositories?per_page=" +
			strconv.Itoa(pageSize)
		if err := c.get(ctx, access, path, &page); err != nil {
			return nil, err
		}

		for _, item := range page.Repositories {
			pushed, _ := time.Parse(repoTimeSlot, item.PushedAt)
			repos = append(repos, domain.Repo{
				FullName: item.FullName, Description: item.Description, Private: item.Private, UpdatedAt: pushed,
			})
		}
	}

	slices.SortStableFunc(repos, func(a, b domain.Repo) int { return b.UpdatedAt.Compare(a.UpdatedAt) })

	return repos[:min(len(repos), maxRepos)], nil
}

// get reads one JSON answer from the API as the person.
func (c *Client) get(ctx context.Context, access, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.api+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+access)

	status, err := c.do(req, out)
	if err != nil {
		return err
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return domain.ErrRejected
	case status != http.StatusOK:
		return fmt.Errorf("%w: %s", errUnavailable, strings.TrimSpace(http.StatusText(status)))
	}

	return nil
}

// CreateRepo makes a repository for the person, with a README, and then adds it to the app's installation so the
// app can clone and push to it. Sharing is best effort: an installation that already covers every repository
// needs nothing, and GitHub says so with a refusal that is not an error here.
func (c *Client) CreateRepo(ctx context.Context, access string, spec domain.NewRepo) (domain.Repo, error) {
	body, _ := json.Marshal(map[string]any{
		"name": spec.Name, "description": spec.Description, "private": spec.Private, "auto_init": true,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api+"/user/repos", bytes.NewReader(body))
	if err != nil {
		return domain.Repo{}, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")

	var made struct {
		FullName string `json:"full_name"`
		ID       int64  `json:"id"`
		Private  bool   `json:"private"`
	}

	status, err := c.do(req, &made)
	if err != nil {
		return domain.Repo{}, err
	}

	switch {
	case status == http.StatusUnprocessableEntity:
		return domain.Repo{}, domain.ErrRepoExists
	case status == http.StatusCreated && made.FullName != "":
	case status == http.StatusUnauthorized:
		return domain.Repo{}, domain.ErrRejected
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return domain.Repo{}, domain.ErrCannotCreate
	default:
		return domain.Repo{}, errUnavailable
	}

	c.share(ctx, access, made.ID)

	return domain.Repo{FullName: made.FullName, Private: made.Private, UpdatedAt: c.now()}, nil
}

// share adds a repository to the app's first installation.
func (c *Client) share(ctx context.Context, access string, repoID int64) {
	var installs struct {
		Installations []struct {
			ID int64 `json:"id"`
		} `json:"installations"`
	}

	err := c.get(ctx, access, "/user/installations?per_page=1", &installs)
	if err != nil || len(installs.Installations) == 0 {
		return
	}

	path := "/user/installations/" + strconv.FormatInt(installs.Installations[0].ID, 10) + "/repositories/" +
		strconv.FormatInt(repoID, 10)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.api+path, http.NoBody)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bearer "+access)

	_, _ = c.do(req, nil)
}
