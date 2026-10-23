package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

type permissionStore struct{ approvalWrites int }

func (*permissionStore) ToolStates(context.Context, string, string) (map[string]bool, error) {
	return nil, nil
}

func (*permissionStore) SetTool(context.Context, string, string, string, bool, time.Time) error {
	return nil
}

func (*permissionStore) SkipApprovals(context.Context, string, string) (bool, error) {
	return false, nil
}

func (s *permissionStore) SetSkipApprovals(context.Context, string, string, bool, time.Time) error {
	s.approvalWrites++
	return nil
}

func TestSetSkipApprovalsRejectsUnsupportedSpace(t *testing.T) {
	store := &permissionStore{}
	registry := NewRegistry("https://example.test",
		domain.Space{Slug: "coding", SupportsApprovals: true},
		domain.Space{Slug: "orchestrator", SupportsApprovals: false},
	)
	permissions := NewPermissions(registry, store, time.Now)

	err := permissions.SetSkipApprovals(
		context.Background(), "user-1", "orchestrator", true,
	)
	if !errors.Is(err, domain.ErrApprovalsUnsupported) {
		t.Fatalf("unsupported approval setting was accepted: %v", err)
	}
	if store.approvalWrites != 0 {
		t.Fatal("unsupported approval setting was persisted")
	}
	if err := permissions.SetSkipApprovals(context.Background(), "user-1", "coding", true); err != nil {
		t.Fatalf("supported approval setting was rejected: %v", err)
	}
}
