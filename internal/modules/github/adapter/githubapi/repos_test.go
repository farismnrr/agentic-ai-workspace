package githubapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// fakeCreate answers the calls a repository creation makes and notes what it saw.
func fakeCreate(created, shared *bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/user/repos":
			body, _ := io.ReadAll(r.Body)
			*created = strings.Contains(string(body), `"auto_init":true`) && strings.Contains(string(body), `"private":true`)

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":42,"full_name":"ann/shop","private":true}`))
		case r.URL.Path == "/user/installations":
			_, _ = w.Write([]byte(`{"installations":[{"id":7}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/user/installations/7/repositories/42":
			*shared = true

			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestANewRepositoryIsCreatedWithAReadmeAndSharedWithTheApp(t *testing.T) {
	t.Parallel()

	var created, shared bool

	c := testClient(t, fakeCreate(&created, &shared))

	repo, err := c.CreateRepo(context.Background(), "tok", domain.NewRepo{Name: "shop", Private: true})
	if err != nil || repo.FullName != "ann/shop" || !repo.Private {
		t.Fatalf("the repository comes back: %+v %v", repo, err)
	}

	if !created || !shared {
		t.Fatalf("created with a README and added to the installation: created=%v shared=%v", created, shared)
	}
}

func TestCreatingARepositoryExplainsWhyItFailed(t *testing.T) {
	t.Parallel()

	for status, want := range map[int]error{
		http.StatusUnprocessableEntity: domain.ErrRepoExists,
		http.StatusForbidden:           domain.ErrCannotCreate,
		http.StatusUnauthorized:        domain.ErrRejected,
	} {
		c := testClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })

		if _, err := c.CreateRepo(context.Background(), "tok", domain.NewRepo{Name: "shop"}); !errors.Is(err, want) {
			t.Errorf("status %d: %v, want %v", status, err, want)
		}
	}
}

func TestEveryPageOfAnInstallationIsRead(t *testing.T) {
	t.Parallel()

	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/installations":
			_, _ = w.Write([]byte(`{"installations":[{"id":7}]}`))
		case "/user/installations/7/repositories":
			_, _ = w.Write([]byte(repoPage(r.URL.Query().Get("page"))))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	repos, err := c.Repos(context.Background(), "tok")
	if err != nil || len(repos) != 150 {
		t.Fatalf("want all 150 repositories, got %d: %v", len(repos), err)
	}

	found := false

	for _, repo := range repos {
		found = found || repo.FullName == "ann/pukat"
	}

	if !found {
		t.Fatal("a repository on the second page was left out")
	}
}

// repoPage is page one (a hundred repositories) or page two (fifty, one of them pukat) of a hundred and fifty.
func repoPage(page string) string {
	names, total := []string{}, 150
	if page == "2" {
		for i := range 49 {
			names = append(names, `{"full_name":"ann/r`+strconv.Itoa(100+i)+`","pushed_at":"2023-01-01T00:00:00Z"}`)
		}

		names = append(names, `{"full_name":"ann/pukat","pushed_at":"2022-01-01T00:00:00Z"}`)
	} else {
		for i := range 100 {
			names = append(names, `{"full_name":"ann/r`+strconv.Itoa(i)+`","pushed_at":"2026-01-01T00:00:00Z"}`)
		}
	}

	return `{"total_count":` + strconv.Itoa(total) + `,"repositories":[` + strings.Join(names, ",") + `]}`
}
