package application

import (
	"encoding/json"
	"strings"
)

// isMutatingGitCall reports whether the git tool call modifies repository files
// or refs on the local workspace. Read-only inspection and push do not need a
// writable local workspace.
func isMutatingGitCall(input json.RawMessage) bool {
	var in struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(input, &in); err != nil || len(in.Args) == 0 {
		return false
	}
	verb := strings.ToLower(strings.TrimSpace(in.Args[0]))
	switch verb {
	case "add", "commit", "fetch", "pull", "merge", "rebase", "checkout", "switch",
		"restore", "reset", "stash", "clean", "mv", "rm", "apply", "cherry-pick", "revert":
		return true
	case "branch", "tag":
		if len(in.Args) > 1 {
			switch in.Args[1] {
			case "-a", "-r", "-v", "-vv", "--list", "--show-current":
				return false
			}
			return true
		}
		return false
	case "remote":
		if len(in.Args) > 1 {
			switch in.Args[1] {
			case "-v", "get-url":
				return false
			}
			return true
		}
		return false
	default:
		return false
	}
}
