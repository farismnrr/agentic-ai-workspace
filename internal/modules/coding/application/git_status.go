package application

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// GitStatusReader reads the current state of one project repository.
type GitStatusReader interface {
	Read() (domain.GitSnapshot, error)
}

// GitHeadReader reads one file's original text from the same project repository.
type GitHeadReader interface {
	ReadHead(path string) (domain.GitHead, error)
}

// GitStatus reads a project's current repository state through its adapter.
type GitStatus struct{ reader GitStatusReader }

// NewGitStatus wires the use case to a Git status adapter.
func NewGitStatus(reader GitStatusReader) *GitStatus {
	return &GitStatus{reader: reader}
}

// Read returns the current bounded repository snapshot.
func (s *GitStatus) Read() (domain.GitSnapshot, error) {
	snapshot, err := s.reader.Read()
	if err != nil {
		return domain.GitSnapshot{}, fmt.Errorf("read project Git status: %w", err)
	}
	return snapshot, nil
}

// GitHead reads one bounded file baseline through its adapter.
type GitHead struct{ reader GitHeadReader }

// NewGitHead wires the use case to a repository adapter.
func NewGitHead(reader GitHeadReader) *GitHead { return &GitHead{reader: reader} }

// Read returns a safe HEAD baseline for one repository-relative file path.
func (s *GitHead) Read(path string) (domain.GitHead, error) {
	head, err := s.reader.ReadHead(path)
	if err != nil {
		return domain.GitHead{}, fmt.Errorf("read project Git HEAD: %w", err)
	}
	return head, nil
}
