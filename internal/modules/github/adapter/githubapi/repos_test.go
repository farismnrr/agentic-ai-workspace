package githubapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

func TestReposAreReadThroughTheInstallationsNewestFirst(t *testing.T) {
	t.Parallel()

	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		switch r.URL.Path {
		case "/user/installations":
			_, _ = w.Write([]byte(`{"installations":[{"id":7}]}`))
		case "/user/installations/7/repositories":
			_, _ = w.Write([]byte(`{"repositories":[
				{"full_name":"ann/old","pushed_at":"2024-01-01T00:00:00Z","private":false},
				{"full_name":"ann/shop","description":"A shop","pushed_at":"2026-05-01T00:00:00Z","private":true}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	repos, err := c.Repos(context.Background(), "tok")
	if err != nil || len(repos) != 2 || repos[0].FullName != "ann/shop" {
		t.Fatalf("newest first: %+v, %v", repos, err)
	}

	if !repos[0].Private || repos[0].Description != "A shop" {
		t.Fatalf("with what the list needs: %+v", repos[0])
	}
}

func TestReposRejectATokenGitHubRefuses(t *testing.T) {
	t.Parallel()

	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })

	if _, err := c.Repos(context.Background(), "wrong"); !errors.Is(err, domain.ErrRejected) {
		t.Fatalf("a token GitHub refuses is rejected: %v", err)
	}
}
