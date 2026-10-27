package application

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxSourceURLLength = 2048

// SaveResearch appends one redacted research round and exports it separately from the PRD.
func (w *Works) SaveResearch(ctx context.Context, userID, client, projectID, workID, markdown string,
	sources []string,
) error {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return err
	}
	document, err := workDocument(markdown, sources)
	if err != nil {
		return err
	}
	work, task, resumed, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return err
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return fmt.Errorf("read research history: %w", err)
	}
	if sameResearch(events, work.ID, document) {
		if resumed {
			return w.appendMutation(ctx, work, nil, nil, nil)
		}
		return nil
	}
	updated := appendResearch(task.Research, document)
	event := workEvent(work, domain.EventResearch, "Research saved", document, w.now())
	taskPatch := &TaskPatch{Research: &updated}
	eventList := []domain.Event{event}
	return w.appendMutation(ctx, work, taskPatch, nil, eventList)
}

// SaveRequirements replaces the requirements document without changing research rounds.
func (w *Works) SaveRequirements(ctx context.Context, userID, client, projectID, workID, markdown string) error {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return err
	}
	document, err := sanitizeText(markdown, domain.MaxWorkMarkdown)
	if err != nil || document == "" {
		return domain.ErrBadRequest
	}
	work, task, resumed, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return err
	}
	if task.PRD == document && !resumed {
		return nil
	}
	event := workEvent(work, domain.EventPRD, "Requirements saved", document, w.now())
	return w.appendMutation(ctx, work, &TaskPatch{PRD: &document}, nil, []domain.Event{event})
}

// Progress records only self-reported evidence from the external assistant.
func (w *Works) Progress(ctx context.Context, userID, client, projectID, workID, message string,
	claims []string,
) error {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return err
	}
	text, err := sanitizeText(message, domain.MaxWorkProgress)
	if err != nil || text == "" || len(claims) > domain.MaxWorkClaims {
		return domain.ErrBadRequest
	}
	cleanClaims := make([]string, 0, len(claims))
	for _, claim := range claims {
		clean, cleanErr := sanitizeText(claim, domain.MaxWorkClaim)
		if cleanErr != nil || clean == "" {
			return domain.ErrBadRequest
		}
		cleanClaims = append(cleanClaims, clean)
	}
	work, _, _, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return err
	}
	technical := strings.Join(cleanClaims, "\n")
	event := workEvent(work, domain.EventMessage, text, technical, w.now())
	event.Work.Evidence = domain.EvidenceClaimed
	events := []domain.Event{event}
	return w.appendMutation(ctx, work, nil, nil, events)
}

func workDocument(markdown string, sources []string) (string, error) {
	document, err := sanitizeText(markdown, domain.MaxWorkMarkdown)
	if err != nil || document == "" || len(sources) > domain.MaxWorkSources {
		return "", domain.ErrBadRequest
	}
	for _, source := range sources {
		clean, cleanErr := sanitizeText(source, maxSourceURLLength)
		if cleanErr != nil || !validSource(clean) {
			return "", domain.ErrBadRequest
		}
		document += "\n\nSource: " + clean
	}
	if utf8.RuneCountInString(document) > domain.MaxWorkMarkdown {
		return "", domain.ErrTooLarge
	}
	return document, nil
}

func sameResearch(events []domain.Event, workID, document string) bool {
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Kind == domain.EventResearch && event.Work != nil && event.Work.ID == workID {
			return requestHash(event.Technical) == requestHash(document)
		}
	}
	return false
}

func validSource(source string) bool {
	parsed, err := url.Parse(source)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Host != "" && parsed.User == nil && !strings.ContainsAny(source, "\r\n")
}
