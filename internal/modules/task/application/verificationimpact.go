package application

import (
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type verificationImpact struct {
	reason       string
	security     bool
	architecture bool
	reviewer     bool
	unknown      bool
}

// assessImpact uses saved task mode and changed paths to select the minimum reviews.
func assessImpact(task domain.Task, events []domain.Event) verificationImpact {
	files := changedPaths(events)
	impact := commandImpact(events)
	var reasons []string
	if impact.reason != "" {
		reasons = append(reasons, impact.reason)
	}
	impact.reason = ""
	architectureAffected := impact.architecture

	for _, name := range files {
		fileImpact := pathImpact(name)
		impact.security = impact.security || fileImpact.security
		impact.unknown = impact.unknown || fileImpact.unknown
		architectureAffected = architectureAffected || fileImpact.architecture
		if fileImpact.reason != "" {
			reasons = append(reasons, fileImpact.reason)
		}
	}

	impact.architecture, impact.reviewer = requiredReviews(
		task.ReviewMode, impact.unknown, architectureAffected, reasons,
	)
	// Security owns architecture verification too, but may only be delegated
	// after the owner explicitly enables Deep Review. Path risk does NOT
	// promote Swift into a mandatory Security review.
	if task.ReviewMode == domain.ReviewModeDeepReview {
		// Deep Review opts into the reviewer whenever implementation files
		// changed, including UI-only work; depth stays impact-proportionate.
		impact.security = impact.security || impact.architecture || hasReviewedChanges(files)
	} else {
		impact.security = false
	}
	impact.reason = strings.Join(slices.Compact(reasons), "; ")
	return impact
}

func hasReviewedChanges(files []string) bool {
	for _, file := range files {
		if !inDocuments(file) {
			return true
		}
	}
	return false
}

func requiredReviews(mode domain.ReviewMode, unknown, architecture bool, reasons []string) (bool, bool) {
	if unknown {
		return true, true
	}
	if mode != domain.ReviewModeDeepReview {
		return false, false
	}
	return architecture, len(reasons) > 0
}

func commandImpact(events []domain.Event) verificationImpact {
	impact := verificationImpact{}
	var reasons []string
	for _, event := range events {
		if event.Run == nil {
			continue
		}
		if !event.Run.EffectsKnown {
			impact.unknown = true
			impact.security = true
			impact.architecture = true
			reasons = append(reasons, "command activity has unrecorded file effects; applying conservative reviews")
			continue
		}
		if len(event.Run.EffectPaths) == 0 && !knownNonMutatingCommand(event.Run.Command) {
			impact.unknown = true
			impact.security = true
			impact.architecture = true
			reasons = append(reasons, "command effects are unchanged but its full command is not a known non-mutating check")
		}
	}
	impact.reason = strings.Join(slices.Compact(reasons), "; ")
	return impact
}

func knownNonMutatingCommand(command string) bool {
	clean := strings.ToLower(strings.TrimSpace(command))
	if clean == "" || strings.ContainsAny(clean, ";|&><`$(){}\n") {
		return false
	}
	for _, safe := range []string{
		"go test", "go vet", "golangci-lint run", "npm run check", "npm run lint", "npm run build",
		"make guardrails", "make backend-test", "make backend-lint", "make verify", "make verify-full",
		"shellcheck", "git status", "git diff", "git log", "git show",
		"pwd", "ls", "cat", "rg", "find", "echo",
	} {
		if clean == safe || strings.HasPrefix(clean, safe+" ") {
			return true
		}
	}
	return false
}

func knownCheckCommand(command string) bool {
	clean := strings.ToLower(strings.TrimSpace(command))
	if clean == "" || strings.ContainsAny(clean, ";|&><`$(){}\n") {
		return false
	}
	for _, check := range []string{
		"go test", "go vet", "golangci-lint run", "npm run check", "npm run lint", "npm run build",
		"make guardrails", "make backend-test", "make backend-lint", "make verify", "make verify-full", "shellcheck",
	} {
		if clean == check || strings.HasPrefix(clean, check+" ") {
			return true
		}
	}
	return false
}
