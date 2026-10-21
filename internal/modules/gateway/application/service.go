// Package application decides which model gateway, if any, a project's sandbox may reach.
package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/domain"
)

// Store keeps the provider each project chose.
type Store interface {
	Get(ctx context.Context, projectID string) (string, error)
	Set(ctx context.Context, projectID, providerID string) error
	Delete(ctx context.Context, projectID string) error
}

// ProjectOwner tells who owns a project.
type ProjectOwner interface {
	Owner(ctx context.Context, projectID string) (userID, slug string, err error)
}

// Providers is what this feature needs from a person's saved model providers.
type Providers interface {
	// Eligible lists the person's enabled OpenAI-compatible gateways.
	Eligible(ctx context.Context, userID string) ([]domain.Choice, error)
	// Credential returns the address and key of one of the person's providers.
	Credential(ctx context.Context, userID, providerID string) (baseURL, apiKey string, err error)
}

// ErrUnset means the project has chosen no gateway.
var ErrUnset = errors.New("no gateway chosen")

// Service lets a project owner choose a gateway and tells the sandbox what that choice allows.
type Service struct {
	store     Store
	owner     ProjectOwner
	providers Providers
}

// NewService builds the use cases.
func NewService(store Store, owner ProjectOwner, providers Providers) *Service {
	return &Service{store: store, owner: owner, providers: providers}
}

// View is the owner's current choice and what they can choose from.
type View struct {
	Selected string          `json:"selected"`
	Options  []domain.Choice `json:"options"`
}

// Get returns the project's choice and the gateways its owner can pick. A choice that is no longer one of them
// (the provider was removed or disabled) shows as none.
func (s *Service) Get(ctx context.Context, userID, projectID string) (View, error) {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return View{}, err
	}

	options, err := s.providers.Eligible(ctx, userID)
	if err != nil {
		return View{}, fmt.Errorf("list gateways: %w", err)
	}

	view := View{Options: options}
	if chosen, gerr := s.store.Get(ctx, projectID); gerr == nil && containsChoice(options, chosen) {
		view.Selected = chosen
	}

	return view, nil
}

// Set lets the project use one of its owner's gateways.
func (s *Service) Set(ctx context.Context, userID, projectID, providerID string) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}

	options, err := s.providers.Eligible(ctx, userID)
	if err != nil {
		return fmt.Errorf("list gateways: %w", err)
	}

	if !containsChoice(options, providerID) {
		return domain.ErrIneligible
	}

	if err = s.store.Set(ctx, projectID, providerID); err != nil {
		return fmt.Errorf("save gateway choice: %w", err)
	}

	return nil
}

// Clear stops the project from using a gateway.
func (s *Service) Clear(ctx context.Context, userID, projectID string) error {
	if err := s.checkOwner(ctx, userID, projectID); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("clear gateway choice: %w", err)
	}

	return nil
}

// Grant returns what the project's sandbox may reach: the gateway's exact host:port and the environment that
// points the code at it. Nothing is granted unless the owner chose it, the provider is still one of theirs and
// still enabled, and its address is on this computer.
func (s *Service) Grant(ctx context.Context, projectID string) (target string, env []string, ok bool) {
	owner, _, err := s.owner.Owner(ctx, projectID)
	if err != nil {
		return "", nil, false
	}

	chosen, err := s.store.Get(ctx, projectID)
	if err != nil {
		return "", nil, false
	}

	options, err := s.providers.Eligible(ctx, owner)
	if err != nil || !containsChoice(options, chosen) {
		return "", nil, false
	}

	baseURL, key, err := s.providers.Credential(ctx, owner, chosen)
	if err != nil {
		return "", nil, false
	}

	host, port, usable := domain.Target(baseURL)
	if !usable {
		return "", nil, false
	}

	return net.JoinHostPort(host, strconv.Itoa(port)), domain.Environment(baseURL, key), true
}

func (s *Service) checkOwner(ctx context.Context, userID, projectID string) error {
	owner, _, err := s.owner.Owner(ctx, projectID)
	if err != nil || owner != userID {
		return domain.ErrNotFound
	}

	return nil
}

func containsChoice(options []domain.Choice, id string) bool {
	return slices.ContainsFunc(options, func(c domain.Choice) bool { return c.ID == id })
}
