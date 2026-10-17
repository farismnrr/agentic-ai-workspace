package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// repoNamePattern is what GitHub accepts for the name part of a repository.
var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

// justCreated is how long a repository the person made here counts as theirs even if GitHub's list has not caught up.
const justCreated = 10 * time.Minute

// Repos lists the repositories the person let the app reach. It needs the connection, and it returns the
// repositories of the person it is asked about only.
func (s *Service) Repos(ctx context.Context, userID string) ([]domain.Repo, error) {
	cred, err := s.Credentials(ctx, userID)
	if err != nil {
		return nil, err
	}

	repos, err := s.provider.Repos(ctx, cred.Token)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	return repos, nil
}

// Reaches reports whether the person let the app reach the repository, or made it here a moment ago.
func (s *Service) Reaches(ctx context.Context, userID, fullName string) (bool, error) {
	if s.madeRecently(userID, fullName) {
		return true, nil
	}

	repos, err := s.Repos(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, repo := range repos {
		if strings.EqualFold(repo.FullName, fullName) {
			return true, nil
		}
	}

	return false, nil
}

// CreateRepo makes a repository on the person's account, with a README so it can be cloned and pushed to at once.
func (s *Service) CreateRepo(ctx context.Context, userID string, spec domain.NewRepo) (domain.Repo, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	if !repoNamePattern.MatchString(spec.Name) || spec.Name == "." || spec.Name == ".." {
		return domain.Repo{}, domain.ErrBadRepoName
	}

	cred, err := s.Credentials(ctx, userID)
	if err != nil {
		return domain.Repo{}, err
	}

	repo, err := s.provider.CreateRepo(ctx, cred.Token, spec)
	if err != nil {
		return domain.Repo{}, fmt.Errorf("create repository: %w", err)
	}

	s.remember(userID, repo.FullName)

	return repo, nil
}

func (s *Service) remember(userID, fullName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.created[userID] == nil {
		s.created[userID] = map[string]time.Time{}
	}

	s.created[userID][strings.ToLower(fullName)] = s.now()
}

func (s *Service) madeRecently(userID, fullName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, ok := s.created[userID][strings.ToLower(fullName)]

	return ok && s.now().Sub(at) < justCreated
}
