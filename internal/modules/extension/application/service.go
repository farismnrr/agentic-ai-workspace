package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

// Deps are the collaborators of Service.
type Deps struct {
	Store    Store
	Packer   Packer
	Projects Projects
	Notify   Notifier
	Now      func() time.Time
}

// Service packages the extensions the AI builds and lists them for the person.
type Service struct{ deps Deps }

// NewService wires the use cases.
func NewService(deps Deps) *Service { return &Service{deps: deps} }

// Result is how a deploy went: the extension when it was packaged, and what to fix when it was not, or to know.
type Result struct {
	// Problems stop the package: the builder must fix them and deploy again.
	Problems []string
	// Notes do not stop it, such as an icon that is missing.
	Notes     []string
	Extension domain.Extension
}

// Deploy packages the extension built into folder (relative to the project) so the person can install it.
func (s *Service) Deploy(ctx context.Context, projectID, folder string) (Result, error) {
	userID, slug, err := s.deps.Projects.Owner(ctx, projectID)
	if err != nil {
		return Result{}, fmt.Errorf("deploy extension: %w", err)
	}

	folder = path.Clean(strings.TrimSpace(folder))

	tree, err := s.deps.Packer.Inspect(ctx, projectID, folder)
	if err != nil {
		return Result{}, fmt.Errorf("deploy extension: %w", err)
	}

	manifest, err := domain.ParseManifest(tree.Manifest)
	if err != nil {
		// A manifest that cannot be read is a problem for the builder to fix, not a failure of the server.
		return Result{Problems: []string{err.Error()}}, nil //nolint:nilerr // reported to the builder
	}

	problems, notes := check(manifest, tree)
	if len(problems) > 0 {
		return Result{Problems: problems, Notes: notes}, nil
	}

	if err = s.deps.Packer.Pack(ctx, projectID, folder); err != nil {
		return Result{}, fmt.Errorf("deploy extension: %w", err)
	}

	extension := domain.Extension{
		DeployedAt: s.deps.Now(), ProjectID: projectID, UserID: userID, Slug: slug, Name: manifest.Name,
		Version: manifest.Version, Description: manifest.Description, Permissions: manifest.Permissions,
		Hosts: append(append([]string{}, manifest.HostPermissions...), matchesOf(manifest)...),
		Size:  tree.Size, Files: tree.Files,
	}
	if err = s.deps.Store.Save(ctx, extension); err != nil {
		return Result{}, fmt.Errorf("deploy extension: %w", err)
	}

	s.deps.Notify.Changed(userID)

	return Result{Extension: extension, Notes: notes}, nil
}

// check finds what stops the package and what only deserves a word.
func check(manifest domain.Manifest, tree Tree) (problems, notes []string) {
	problems = manifest.Problems()

	for _, file := range manifest.Referenced() {
		if !tree.Has(file) {
			problems = append(problems, "manifest.json points at "+file+", which is not in the folder")
		}
	}

	if len(manifest.Icons) == 0 {
		notes = append(notes, "no icons in manifest.json: the browser will show a plain letter instead")
	}

	return problems, notes
}

func matchesOf(manifest domain.Manifest) []string {
	var out []string
	for _, script := range manifest.ContentScripts {
		out = append(out, script.Matches...)
	}

	return out
}

// List returns the person's extensions, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]domain.Extension, error) {
	list, err := s.deps.Store.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list extensions: %w", err)
	}

	return list, nil
}

// Package opens the package of one of the person's extensions.
func (s *Service) Package(
	ctx context.Context, userID, projectID string,
) (domain.Extension, io.ReadCloser, int64, error) {
	extension, err := s.deps.Store.ByProject(ctx, userID, projectID)
	if err != nil {
		return domain.Extension{}, nil, 0, fmt.Errorf("find extension: %w", err)
	}

	file, size, err := s.deps.Packer.Open(projectID)
	if err != nil {
		return domain.Extension{}, nil, 0, fmt.Errorf("open package: %w", err)
	}

	return extension, file, size, nil
}

// Remove deletes one of the person's extensions and its package.
func (s *Service) Remove(ctx context.Context, userID, projectID string) error {
	if err := s.deps.Store.Delete(ctx, userID, projectID); err != nil {
		return fmt.Errorf("remove extension: %w", err)
	}

	if err := s.deps.Packer.Remove(projectID); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("remove package: %w", err)
	}

	s.deps.Notify.Changed(userID)

	return nil
}

// RemoveAll deletes every extension of a person, when their account is deleted.
func (s *Service) RemoveAll(ctx context.Context, userID string) error {
	list, err := s.deps.Store.List(ctx, userID)
	if err != nil {
		return fmt.Errorf("list extensions: %w", err)
	}

	for _, extension := range list {
		if err = s.Remove(ctx, userID, extension.ProjectID); err != nil {
			return err
		}
	}

	return nil
}

// Forget removes a project's extension when the project is archived, and tells the owner's pages.
func (s *Service) Forget(ctx context.Context, projectID string) {
	userID, err := s.deps.Store.DeleteProject(ctx, projectID)
	if err != nil || userID == "" {
		return
	}

	_ = s.deps.Packer.Remove(projectID)

	s.deps.Notify.Changed(userID)
}
