package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxPlanDecisionNote = 500

// PlanReviewInput identifies the exact external plan revision being reviewed.
type PlanReviewInput struct {
	ProjectID      string
	TaskID         string
	WorkID         string
	IdempotencyKey string
	Note           string
	Revision       int64
}

// PlanReviews owns the app and MCP decisions for external plan revisions.
type PlanReviews struct {
	store    PlanDecisionStore
	ids      IDs
	now      func() time.Time
	kicker   HandoffKicker
	notifier PlanReviewNotifier
}

// NewPlanReviews wires decision persistence and post-commit handoff delivery.
func NewPlanReviews(store PlanDecisionStore, ids IDs, now func() time.Time,
	kicker HandoffKicker, notifier PlanReviewNotifier,
) *PlanReviews {
	return &PlanReviews{store: store, ids: ids, now: now, kicker: kicker, notifier: notifier}
}

// ApproveFromApp records a browser decision using the authenticated session owner.
func (p *PlanReviews) ApproveFromApp(ctx context.Context, userID string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	return p.approve(ctx, userID, "", "app", in)
}

// ApproveFromMCP records a connected app decision using the verified token identity.
func (p *PlanReviews) ApproveFromMCP(ctx context.Context, userID, client string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	return p.approve(ctx, userID, client, "mcp", in)
}

// RequestChanges returns the submitted revision to the planner without creating a handoff.
func (p *PlanReviews) RequestChanges(ctx context.Context, userID string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	result, err := p.decide(ctx, userID, "", "app", domain.PlanDecisionChangesRequested, in)
	if err == nil && p.notifier != nil {
		p.notifier.Changed(userID, in.TaskID)
	}
	return result, err
}

func (p *PlanReviews) approve(ctx context.Context, userID, client, source string, in PlanReviewInput) (
	PlanDecisionResult, error,
) {
	result, err := p.decide(ctx, userID, client, source, domain.PlanDecisionApproved, in)
	if err != nil {
		return PlanDecisionResult{}, err
	}
	if p.notifier != nil {
		p.notifier.Changed(userID, in.TaskID)
	}
	if p.kicker != nil && result.HandoffID != "" && result.HandoffState != "done" {
		taskID, kickErr := p.kicker.Kick(ctx, result.HandoffID)
		if taskID != "" && kickErr == nil {
			result.InternalTaskID, result.HandoffState = taskID, "done"
		}
	}
	return result, nil
}

func (p *PlanReviews) decide(ctx context.Context, userID, client, source string,
	decision domain.PlanDecisionKind, in PlanReviewInput,
) (PlanDecisionResult, error) {
	if err := validatePlanReview(userID, in); err != nil {
		return PlanDecisionResult{}, err
	}
	note := strings.TrimSpace(in.Note)
	if note != "" {
		note, _ = domain.Redact(note)
	}
	requestHash, err := planDecisionHash(decision, in.Revision, note)
	if err != nil {
		return PlanDecisionResult{}, err
	}
	decisionID, err := p.ids.ID()
	if err != nil {
		return PlanDecisionResult{}, fmt.Errorf("new plan decision id: %w", err)
	}
	write := PlanDecisionWrite{Decision: domain.PlanDecision{
		ID: decisionID, WorkID: in.WorkID, TaskID: in.TaskID, UserID: userID, ProjectID: in.ProjectID,
		Kind: decision, Source: source, Client: client, IdemKey: in.IdempotencyKey,
		ReqHash: requestHash, Note: note, DecidedAt: p.clock(),
	}, Revision: in.Revision}
	if decision == domain.PlanDecisionApproved {
		write.HandoffID, err = p.ids.ID()
		if err != nil {
			return PlanDecisionResult{}, fmt.Errorf("new plan handoff id: %w", err)
		}
	}
	result, err := p.store.DecidePlan(ctx, write)
	if err != nil {
		return PlanDecisionResult{}, fmt.Errorf("record plan decision: %w", err)
	}
	return result, nil
}

func validatePlanReview(userID string, in PlanReviewInput) error {
	if userID == "" {
		return domain.ErrUnverifiedUser
	}
	if in.ProjectID == "" || in.TaskID == "" || in.WorkID == "" || in.Revision < 1 ||
		in.IdempotencyKey == "" || !validKey(in.IdempotencyKey) ||
		utf8.RuneCountInString(in.Note) > maxPlanDecisionNote {
		return domain.ErrBadRequest
	}
	return nil
}

func (p *PlanReviews) clock() time.Time {
	if p.now == nil {
		return time.Now().UTC()
	}
	return p.now().UTC()
}
