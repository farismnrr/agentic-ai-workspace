package domain

import (
	"path"
	"strings"
)

// GitPath validates a canonical repository-relative path without normalizing traversal.
func GitPath(raw string) (string, error) {
	if raw == "" || len(raw) > MaxQueryLength || strings.ContainsRune(raw, 0) ||
		strings.Contains(raw, `\`) || strings.HasPrefix(raw, "/") || path.Clean(raw) != raw || Denied(raw) {
		return "", ErrDenied
	}
	for _, segment := range strings.Split(raw, "/") {
		if segment == "." || segment == ".." || segment == "" {
			return "", ErrDenied
		}
	}
	return raw, nil
}

// GitSnapshot is a bounded view of paths changed in one project's repository.
type GitSnapshot struct {
	Available bool      `json:"available"`
	Branch    string    `json:"branch,omitempty"`
	Revision  string    `json:"revision,omitempty"`
	Initial   bool      `json:"initial"`
	Files     []GitFile `json:"files"`
}

// GitFile retains index, worktree, conflict, and submodule state separately.
type GitFile struct {
	Path      string `json:"path"`
	Original  string `json:"originalPath,omitempty"`
	Index     string `json:"index"`
	Worktree  string `json:"worktree"`
	Conflict  bool   `json:"conflict"`
	Untracked bool   `json:"untracked"`
	Submodule bool   `json:"submodule"`
}

// GitHead is a bounded text baseline from the project's current HEAD commit.
type GitHead struct {
	Available bool   `json:"available"`
	Initial   bool   `json:"initial"`
	Content   string `json:"content"`
	Binary    bool   `json:"binary,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}
