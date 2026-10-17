package githubapi

import (
	"context"
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
