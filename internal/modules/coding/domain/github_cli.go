package domain

import (
	"fmt"
	"strings"
)

// GitHubTask supports the documented GitHub CLI top-level families without
// a shell, extension/alias dispatch or unrestricted API mutation.
// role is assigned by the server, never by the tool caller.
func GitHubTask(role string, args []string) (Task, error) {
	if len(args) > 0 && args[0] == "gh" {
		args = args[1:]
	}
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "" {
			cleaned = append(cleaned, arg)
		}
	}
	args = cleaned
	if err := vcsArgs(args); err != nil {
		return Task{}, err
	}
	family := args[0]
	if len(args) == 1 && family == "status" {
		return Task{Name: "gh", Kind: "git", Description: "GitHub status", Command: []string{"gh", "status"},
			TimeoutSeconds: 180}, nil
	}
	if len(args) < 2 {
		return Task{}, ErrBadGitArgs
	}
	action := args[1]
	if dangerousGhOptions(args) {
		return Task{}, fmt.Errorf("%w: GitHub command flags require separate approval", ErrBadGitArgs)
	}

	// Permissions describe operations, not a short list of GitHub versions.
	// GitHub itself remains the authority for which repositories are granted.
	read := false
	writeDeveloper := false
	writeQA := false
	writeDevOps := false
	switch family {
	case "issue":
		read = ghContains(action, "list view status")
		writeDeveloper = ghContains(action, "create edit comment close reopen transfer pin unpin develop")
	case "pr":
		read = ghContains(action, "list view status checks diff")
		writeDeveloper = ghContains(action, "create edit comment checkout ready update-branch")
		writeQA = action == "review"
	case "run":
		read = ghContains(action, "list view watch download")
		writeDevOps = action == "rerun"
	case "workflow":
		read = ghContains(action, "list view")
		writeDevOps = action == "run"
	case "release":
		read = ghContains(action, "list view download verify verify-asset")
		writeDevOps = ghContains(action, "create upload edit")
	case "repo":
		read = ghContains(action, "list view")
		writeDeveloper = ghContains(action, "clone fork create sync")
	case "project":
		read = ghContains(action, "list view field-list item-list status-list")
		writeDeveloper = ghContains(action, "create copy edit close reopen item-add item-edit item-archive item-restore field-create field-edit")
	case "discussion":
		read = ghContains(action, "list view")
	case "search":
		read = ghContains(action, "code commits issues prs repos")
	case "label":
		read = ghContains(action, "list")
		writeDeveloper = ghContains(action, "create edit")
	case "gist":
		read = ghContains(action, "list view")
		writeDeveloper = ghContains(action, "create edit")
	case "api":
		// Arbitrary POST/PATCH/DELETE and GraphQL mutations require an
		// independent approval path; read-only REST GET works with gh api.
		read = ghAPIReadOnly(args)
	case "auth":
		read = action == "status"
	case "config":
		read = action == "get" || action == "list"
	case "status":
		read = true
	}
	if !read {
		switch role {
		case "developer":
			if !writeDeveloper {
				return Task{}, ErrBadGitArgs
			}
		case "qa":
			if !writeQA {
				return Task{}, ErrBadGitArgs
			}
		case "devops":
			if !writeDevOps {
				return Task{}, ErrBadGitArgs
			}
		default:
			return Task{}, ErrBadGitArgs
		}
	}
	return Task{Name: "gh", Kind: "git", Description: "GitHub CLI " + family + " " + action,
		Command: append([]string{"gh"}, args...), TimeoutSeconds: 180}, nil
}

func ghContains(v, options string) bool {
	for _, option := range strings.Fields(options) {
		if v == option {
			return true
		}
	}
	return false
}

func ghAPIReadOnly(args []string) bool {
	if len(args) < 2 || args[1] == "graphql" ||
		strings.HasSuffix(args[1], "/graphql") {
		// GraphQL mutation semantics cannot be inferred from HTTP method.
		return false
	}
	for i, arg := range args[1:] {
		if arg == "--method" || arg == "-X" {
			at := i + 2
			if at >= len(args) || args[at] != "GET" {
				return false
			}
		}
		if strings.HasPrefix(arg, "--method=") && arg != "--method=GET" {
			return false
		}
		if strings.HasPrefix(arg, "-X") && len(arg) > 2 && arg != "-XGET" {
			return false
		}
		for _, changing := range []string{"-f", "-F", "--field", "--raw-field", "--input", "--hostname", "--header"} {
			if arg == changing || strings.HasPrefix(arg, changing+"=") ||
				((changing == "-f" || changing == "-F") && strings.HasPrefix(arg, changing) && len(arg) > 2) {
				return false
			}
		}
	}
	return true
}

func dangerousGhOptions(args []string) bool {
	for _, arg := range args[1:] {
		switch arg {
		case "--admin", "--delete", "--force", "--clobber", "--yes",
			"-H", "--verbose", "--debug",
			"--confirm", "--host", "--hostname", "--repo-delete", "--shell",
			"--editor", "--web", "--recover", "--input":
			return true
		}
		for _, forbidden := range []string{"--host=", "--hostname=", "--repo-delete=",
			"--editor=", "--input=", "--force=", "--shell=",
			"--header="} {
			if strings.HasPrefix(arg, forbidden) {
				return true
			}
		}
	}
	return false
}
