package domain

import (
	"errors"
	"testing"
	"time"
)

func TestOnlyAnOwnerOrAdminDecides(t *testing.T) {
	g := pending(t)

	for _, who := range []Decider{nobody, {}} {
		if _, err := g.Approve(who, 0, t0); !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("approve by %+v: %v", who, err)
		}

		if _, err := g.Deny(who, t0); !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("deny by %+v: %v", who, err)
		}
	}

	for _, who := range []Decider{owner, admin} {
		if _, err := g.Approve(who, 0, t0); err != nil {
			t.Errorf("approve by %+v: %v", who, err)
		}
	}
}

func TestADecidedOrExpiredRequestCannotBeDecidedAgain(t *testing.T) {
	approved, _ := pending(t).Approve(owner, 0, t0)
	denied, _ := pending(t).Deny(owner, t0)
	revoked, _ := approved.Revoke(owner, t0)

	for name, g := range map[string]Grant{"approved": approved, "denied": denied, "revoked": revoked} {
		if _, err := g.Approve(owner, 0, t0); !errors.Is(err, ErrNotPending) {
			t.Errorf("%s approved again: %v", name, err)
		}

		if _, err := g.Deny(owner, t0); !errors.Is(err, ErrNotPending) {
			t.Errorf("%s denied again: %v", name, err)
		}
	}

	if _, err := pending(t).Approve(owner, 0, t0.Add(PendingTTL)); !errors.Is(err, ErrNotPending) {
		t.Errorf("an unanswered request must stop being approvable: %v", err)
	}
}

func TestRevokeEndsAGrantAtOnce(t *testing.T) {
	approved, _ := pending(t).Approve(owner, 0, t0)
	now := t0.Add(time.Hour)

	if _, err := approved.Revoke(nobody, now); !errors.Is(err, ErrNotAuthorized) {
		t.Errorf("revoke by nobody: %v", err)
	}

	revoked, err := approved.Revoke(admin, now)
	if err != nil || revoked.Active(now) || revoked.Status != StatusRevoked {
		t.Fatalf("revoked %+v, %v", revoked, err)
	}

	if _, err = revoked.Revoke(owner, now); !errors.Is(err, ErrNotApproved) {
		t.Errorf("revoking twice: %v", err)
	}

	if _, err = pending(t).Revoke(owner, now); !errors.Is(err, ErrNotApproved) {
		t.Errorf("revoking something never approved: %v", err)
	}
}

func TestAnOwnerCannotDecideOnAnotherProjectsGrant(t *testing.T) {
	g := pending(t)
	approved, _ := g.Approve(owner, 0, t0)

	for _, who := range []Decider{
		{ID: "u1", ProjectID: "p2", Owner: true}, {ID: "u1", Owner: true}, {ID: "", ProjectID: "p1", Owner: true},
	} {
		if _, err := g.Approve(who, 0, t0); !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("approve by %+v: %v", who, err)
		}

		if _, err := g.Deny(who, t0); !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("deny by %+v: %v", who, err)
		}

		if _, err := approved.Revoke(who, t0); !errors.Is(err, ErrNotAuthorized) {
			t.Errorf("revoke by %+v: %v", who, err)
		}
	}

	if _, err := g.Approve(Decider{ID: "root", Admin: true}, 0, t0); err != nil {
		t.Errorf("an admin spans projects: %v", err)
	}
}
