package domain

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
