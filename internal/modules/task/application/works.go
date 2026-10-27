package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Works contains the use cases for a connected app's explicit work session.
type Works struct {
	deps    Deps
	store   WorkStore
	limits  workLimiter
	beginMu sync.Mutex
}

// NewWorks wires external work use cases to their persistence boundary.
func NewWorks(deps Deps, store WorkStore) *Works {
	return &Works{deps: deps, store: store, limits: workLimiter{buckets: map[string]map[int64]int{}}}
}

// BeginInput is the request supplied by a connected app after token verification.
type BeginInput struct {
	UserID         string
	Client         string
	ProjectID      string
	Summary        string
	IdempotencyKey string
	PlannedRoles   []string
}

// WorkRole is the public, prompt-free part of an enabled team role.
type WorkRole struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
}

type beginMode uint8

const (
	beginWithoutKey beginMode = iota
	beginWithKey
	beginResumed
)

// Begin starts a work session without merging it into the legacy 30-minute thread.
func (w *Works) Begin(ctx context.Context, in BeginInput) (BeginResult, error) {
	if err := w.limits.allow(in.UserID, true, w.now()); err != nil {
		return BeginResult{}, err
	}
	clean, planned, hash, roster, err := w.prepareBegin(ctx, in)
	if err != nil {
		return BeginResult{}, err
	}

	w.beginMu.Lock()
	defer w.beginMu.Unlock()
	prior, resumed, err := w.resume(ctx, in, hash)
	if err != nil {
		return BeginResult{}, err
	}
	if resumed {
		return w.beginResult(prior, true, roster), nil
	}
	if err := w.abandonStale(ctx, in.UserID); err != nil {
		return BeginResult{}, err
	}
	if err := w.checkCaps(ctx, in.UserID, in.ProjectID); err != nil {
		return BeginResult{}, err
	}
	return w.create(ctx, in, clean, hash, planned, roster)
}

func (w *Works) prepareBegin(
	ctx context.Context, in BeginInput,
) (string, []string, string, Roster, error) {
	summary := strings.TrimSpace(in.Summary)
	if err := validateBeginInput(in, summary); err != nil {
		return "", nil, "", Roster{}, domain.ErrBadRequest
	}
	if _, err := w.deps.Projects.Name(ctx, in.UserID, in.ProjectID); err != nil {
		return "", nil, "", Roster{}, domain.ErrNoProject
	}
	roster, err := w.roster(ctx, in.UserID)
	if err != nil {
		return "", nil, "", Roster{}, err
	}
	planned, err := normalizePlanned(in.PlannedRoles, roster)
	if err != nil {
		return "", nil, "", Roster{}, err
	}
	clean, _ := domain.Redact(summary)
	hash, err := beginHash(clean, planned)
	return clean, planned, hash, roster, err
}

func validateBeginInput(in BeginInput, summary string) error {
	if in.UserID == "" || strings.TrimSpace(in.Client) == "" || strings.TrimSpace(in.ProjectID) == "" ||
		summary == "" || utf8.RuneCountInString(summary) > domain.MaxWorkSummary ||
		len(in.PlannedRoles) > domain.MaxPlannedRoles || !validKey(in.IdempotencyKey) {
		return domain.ErrBadRequest
	}
	return nil
}

func (w *Works) resume(ctx context.Context, in BeginInput, hash string) (domain.ExternalWork, bool, error) {
	if in.IdempotencyKey == "" {
		return domain.ExternalWork{}, false, nil
	}
	prior, found, err := w.store.FindIdempotent(ctx, in.UserID, in.Client, in.ProjectID, in.IdempotencyKey)
	if err != nil {
		return domain.ExternalWork{}, false, fmt.Errorf("check idempotent work: %w", err)
	}
	if !found {
		return domain.ExternalWork{}, false, nil
	}
	if prior.IdemHash != hash {
		return domain.ExternalWork{}, false, domain.ErrKeyReused
	}
	return prior, true, nil
}

func (w *Works) create(
	ctx context.Context, in BeginInput, summary, hash string, planned []string, roster Roster,
) (BeginResult, error) {
	taskID, err := w.deps.IDs.ID()
	if err != nil {
		return BeginResult{}, fmt.Errorf("new external task id: %w", err)
	}
	workID, err := w.deps.IDs.ID()
	if err != nil {
		return BeginResult{}, fmt.Errorf("new work id: %w", err)
	}
	now := w.now()
	task := domain.Task{
		ID: taskID, UserID: in.UserID, ProjectID: in.ProjectID, Source: in.Client,
		Title: titleOf(summary), Summary: summary, Status: domain.StatusWorking,
		WorkID: workID, CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{},
	}
	work := domain.ExternalWork{
		ID: workID, TaskID: taskID, UserID: in.UserID, ProjectID: in.ProjectID, Client: in.Client,
		State: domain.WorkOpen, IdemKey: in.IdempotencyKey, IdemHash: hash,
		CreatedAt: now, UpdatedAt: now, NextStep: 1,
	}
	event := domain.Event{
		Kind: domain.EventRequest, Text: summary, Time: now,
		Work: &domain.WorkRef{ID: workID, Actor: in.Client, PlannedRoles: planned},
	}
	created, err := w.store.CreateWork(ctx, WorkCreate{Task: task, Work: work, Event: event})
	if err != nil {
		return BeginResult{}, fmt.Errorf("create external work: %w", err)
	}
	if created.ID != workID {
		if created.IdemHash != hash {
			return BeginResult{}, domain.ErrKeyReused
		}
		return w.beginResult(created, true, roster), nil
	}
	w.notify(ctx, task)
	return w.beginResult(created, false, roster), nil
}

func (w *Works) now() time.Time {
	if w.deps.Now != nil {
		return w.deps.Now()
	}
	return time.Now()
}

func beginHash(summary string, planned []string) (string, error) {
	data, err := json.Marshal(struct {
		Summary string
		Roles   []string
	}{summary, planned})
	if err != nil {
		return "", fmt.Errorf("encode begin request: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func validKey(key string) bool {
	if key == "" {
		return true
	}
	return utf8.RuneCountInString(key) >= domain.MinRequestID &&
		utf8.RuneCountInString(key) <= domain.MaxRequestID
}
