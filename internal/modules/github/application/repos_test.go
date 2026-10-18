package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

func TestOnlyARealRepositoryNameIsSentToGitHub(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")

	ctx := context.Background()

	for _, bad := range []string{"", " ", "a b", "a/b", "..", ".", "x; rm -rf /", "é"} {
		if _, err := r.svc.CreateRepo(ctx, "alice", domain.NewRepo{Name: bad}); !errors.Is(err, domain.ErrBadRepoName) {
			t.Errorf("%q must be refused: %v", bad, err)
		}
	}

	repo, err := r.svc.CreateRepo(ctx, "alice", domain.NewRepo{Name: "my-shop.v2", Private: true})
	if err != nil || repo.FullName != "ann/my-shop.v2" {
		t.Fatalf("a good name goes through: %+v %v", repo, err)
	}

	if ok, _ := r.svc.Reaches(ctx, "alice", "ANN/my-shop.v2"); !ok {
		t.Fatal("a repository made here a moment ago counts as reachable, whatever its case")
	}

	if ok, _ := r.svc.Reaches(ctx, "alice", "ann/other"); ok {
		t.Fatal("a repository that was not listed and not made here is not reachable")
	}
}
