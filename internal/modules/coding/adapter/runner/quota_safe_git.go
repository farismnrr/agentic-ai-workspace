package runner

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// quotaSafeGit marks only server-authorized Git operations that do not need
// to write to the project. Large generated/ignored files must not prevent an
// owner from inspecting or pushing an existing commit. The runner mounts the
// whole workspace read-only for these operations and never gives shell tasks
// this exception. Write-capable Git, such as fetch/commit/checkout, is capped.
func quotaSafeGit(ctx context.Context, task domain.Task) bool {
	if !domain.GitOnly(ctx) || domain.WritableGitWorkspace(ctx) {
		return false
	}

	args := task.Command
	if task.Kind == "git" {
		// GitTask, not arbitrary model-supplied shell argv, creates this prefix.
		if len(args) < 5 || args[0] != "git" || args[1] != "-c" ||
			args[2] != "core.hooksPath=/dev/null" || args[3] != "--no-pager" {
			return false
		}
		switch args[4] {
		case "status", "rev-parse", "ls-remote", "ls-files", "show-ref", "merge-base":
			return true
		case "diff":
			// GitTask forces --no-ext-diff for diff to prevent repository
			// configured diff helpers from executing with GitHub credentials.
			return len(args) >= 6 && args[5] == "--no-ext-diff"
		case "log":
			// Commit inspection only. Patch/textconv modes remain quota-gated.
			for _, arg := range args[5:] {
				if arg == "-p" || arg == "--patch" || arg == "--textconv" ||
					arg == "--ext-diff" || arg == "--remerge-diff" {
					return false
				}
			}
			return true
		case "branch":
			// Listing branches is safe; create, rename, delete and
			// upstream operations must still obey workspace quota.
			if len(args) == 5 {
				return true
			}
			switch args[5] {
			case "-a", "-r", "-v", "-vv", "--list", "--show-current":
				return len(args) == 6
			}
			return false
		case "push":
			// Remote writes are subject to the existing role, credentials and
			// dangerousGit guards. The local project remains read-only.
			return true
		case "remote":
			// Remote mutations (add, set-url, rename, remove) remain capped.
			return len(args) == 5 ||
				(len(args) == 6 && args[5] == "-v") ||
				(len(args) == 7 && args[5] == "get-url")
		}
		return false
	}

	// gh may inspect the connected account even if project build artifacts
	// exceed the workspace quota, but no GitHub mutation gets an exemption.
	return task.Name == "gh" && len(args) == 3 &&
		args[0] == "gh" && args[1] == "auth" && args[2] == "status"
}
