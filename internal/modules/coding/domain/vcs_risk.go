package domain

import "strings"

// VCSReadOnly is used for low-friction internal approvals. Every mutation or
// unknown operation keeps the normal approval path. The executor still makes
// the final server-owned role/operation decision separately.
func VCSReadOnly(program string, args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, arg := range args[1:] {
		if arg == "-o" || arg == "--output" || strings.HasPrefix(arg, "--output=") {
			return false
		}
	}
	verb := args[0]
	switch program {
	case "git":
		switch verb {
		case "status", "log", "show", "blame", "diff", "rev-list", "rev-parse",
			"ls-files", "ls-tree", "ls-remote", "show-ref", "shortlog",
			"merge-base", "describe", "count-objects", "cat-file",
			"verify-commit", "verify-tag", "check-ignore", "check-attr":
			return true
		case "branch", "worktree":
			return len(args) == 1 || (len(args) > 1 && ghContains(args[1], "--list -l list"))
		case "config":
			return len(args) > 1 && ghContains(args[1], "--list --get --get-all -l")
		}
	case "gh":
		if len(args) < 2 {
			return false
		}
		switch verb {
		case "issue", "pr", "workflow", "release", "repo", "project", "discussion", "gist", "label":
			return ghContains(args[1], "list view status checks diff field-list item-list status-list")
		case "run":
			return ghContains(args[1], "list view watch")
		case "search":
			return true
		case "api":
			return ghAPIReadOnly(args)
		case "auth":
			return args[1] == "status"
		}
	}
	return false
}
