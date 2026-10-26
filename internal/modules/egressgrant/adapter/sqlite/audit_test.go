package sqlite_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/sqlite"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

func TestGrantAuditIsAtomicAndContainsNoReasonOrSecret(t *testing.T) {
	w := newWorld(t)
	secret := "password=do-not-ever-log"
	g, err := w.Request(t.Context(), "p1", "t1", "audit.example.com", secret, agent)
	if err != nil {
		t.Fatal(err)
	}
	w.approve(t, g, p1own, time.Hour)
	if _, err = w.Revoke(t.Context(), g.ID, p1own); err != nil {
		t.Fatal(err)
	}
	var count int
	var entries string
	err = w.db.QueryRowContext(t.Context(), `SELECT COUNT(*),group_concat(tool||':'||summary||':'||detail, ' | ')
 FROM activity WHERE client = ?`, g.ID).Scan(&count, &entries)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || strings.Contains(entries, secret) || strings.Contains(entries, "password=") {
		t.Fatalf("incomplete or sensitive audit: %d %q", count, entries)
	}
	var other int
	err = w.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM activity WHERE client=? AND user_id!='u1'`, g.ID).
		Scan(&other)
	if err != nil || other != 0 {
		t.Fatalf("activity leaked outside project owner: %d %v", other, err)
	}
}

func TestExpiredGrantAuditIsIdempotentAcrossRuns(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "expiry.example.com")
	audit := sqlite.NewAudit(w.db)
	for i := range 3 {
		n, err := audit.Expired(t.Context(), t0.Add(domain.PendingTTL+time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if (i == 0 && n != 1) || (i > 0 && n != 0) {
			t.Fatalf("sweep %d inserted %d", i, n)
		}
	}
	var count int
	if err := w.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM activity WHERE client=? AND tool='egress_expired'`, g.ID).
		Scan(&count); err != nil || count != 1 {
		t.Fatalf("expiry events %d %v", count, err)
	}
}

func TestFinishedTaskCannotBeApprovedEvenBeforeSweep(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "finished.example.com")
	w.setStatus(t, "t1", "done")
	_, err := w.Approve(t.Context(), g.ID, p1own, time.Hour)
	if !errors.Is(err, application.ErrScope) {
		t.Fatalf("approved ended task: %v", err)
	}
	var status string
	if err := w.db.QueryRowContext(t.Context(), `SELECT status FROM egress_grants WHERE id=?`, g.ID).
		Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("unexpected mutation: %s", status)
	}
}

func TestEnforcementIncidentDoesNotExposeGrantReason(t *testing.T) {
	w := newWorld(t)
	g := w.ask(t, "p1", "t1", "incident.example.com")
	audit := sqlite.NewAudit(w.db)
	if err := audit.EnforcementFailure(context.Background(), g.ID); err != nil {
		t.Fatal(err)
	}
	var summary, detail, outcome string
	err := w.db.QueryRowContext(t.Context(), `SELECT summary,detail,outcome FROM activity
 WHERE client=? AND tool='egress_enforcement_failed'`, g.ID).Scan(&summary, &detail, &outcome)
	if err != nil || detail != "" || outcome != "failed" || !strings.Contains(summary, "incident.example.com") {
		t.Fatalf("incident %s %s %s %v", summary, detail, outcome, err)
	}
}
