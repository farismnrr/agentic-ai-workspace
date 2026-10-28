package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"

// GitStatusReader reads the current state of one project repository.
type GitStatusReader interface {
	Read() (domain.GitSnapshot, error)
}

// GitStatus reads a project's current repository state through its adapter.
type GitStatus struct{ reader GitStatusReader }

// NewGitStatus wires the use case to a Git status adapter.
func NewGitStatus(reader GitStatusReader) *GitStatus {
	return &GitStatus{reader: reader}
}

// Read returns the current bounded repository snapshot.
func (s *GitStatus) Read() (domain.GitSnapshot, error) {
	return s.reader.Read()
}
