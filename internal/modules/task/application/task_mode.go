package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	deepReviewDirective = "@deep-review"
	previewLength       = 1200
)

func requestedReviewMode(requested, text string, current domain.ReviewMode) (domain.ReviewMode, error) {
	mode := domain.ReviewMode(requested)
	if requested != "" && !mode.Valid() {
		return "", domain.ErrBadMode
	}
	if current == domain.ReviewModeDeepReview && mode == domain.ReviewModeSwift {
		return "", domain.ErrBadMode
	}
	if mode == "" {
		mode = current
	}
	if mode == "" {
		mode = domain.ReviewModeSwift
	}
	if hasDeepReviewDirective(text) {
		mode = domain.ReviewModeDeepReview
	}
	return mode, nil
}

func hasDeepReviewDirective(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return line == deepReviewDirective
	}
	return false
}

func (s *Service) prepareOwnerRequest(
	ctx context.Context, taskID, projectID, text string,
) (domain.Event, string, error) {
	prompt, files, err := s.inputDocumentForProject(ctx, taskID, projectID, text)
	if err != nil {
		return domain.Event{}, "", err
	}
	event := domain.Event{Documents: files, Kind: domain.EventRequest, Text: text, Time: s.deps.Now()}
	if len(files) > 0 {
		event.Text = "Long message saved as a project document.\n\n" + truncate(text, previewLength)
		event.Technical = prompt
	}
	return event, prompt, nil
}
