package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBadGitArgs means the requested Git/GitHub operation does not fit the
// bounded developer/QA/DevOps command interface.
var ErrBadGitArgs = errors.New("Git or GitHub CLI arguments are unsupported or need separate approval")

// VCS commands are executable binaries, never shell snippets. Git's external
// executable subcommand mechanism means the recognized Git command names are
// enumerated; supporting a new Git release only needs updating this registry.
var gitBuiltins = strings.Fields(`
add am annotate apply archive bisect blame branch bugreport bundle cat-file check-attr
check-ignore check-mailmap check-ref-format checkout checkout-index cherry cherry-pick
citool clean clone column commit commit-graph config count-objects credential
describe diff diff-files diff-index diff-tree fetch fetch-pack fmt-merge-msg
for-each-ref for-each-repo format-patch fsck gc get-tar-commit-id grep
hash-object help index-pack init interpret-trailers log ls-files ls-remote
ls-tree mailinfo mailsplit maintenance merge merge-base merge-file merge-index
merge-one-file merge-tree mktag mktree multi-pack-index mv name-rev notes
pack-objects pack-redundant pack-refs patch-id prune prune-packed pull push
range-diff read-tree rebase receive-pack reflog remote repack replace
rerere reset restore revert rev-list rev-parse rm send-email shortlog
show show-branch show-index show-ref sparse-checkout stash status
stripspace submodule switch symbolic-ref tag unpack-file unpack-objects
update-index update-ref update-server-info upload-archive upload-pack
var verify-commit verify-pack verify-tag version web--browse whatchanged
worktree write-tree
`)

// GitTask executes a recognized native Git command using direct argv.
// It retains broad Git workflows without opening a shell or custom Git alias.
func GitTask(args []string) (Task, error) {
	if err := vcsArgs(args); err != nil {
		return Task{}, err
	}
	verb := args[0]
	found := false
	for _, name := range gitBuiltins {
		if verb == name {
			found = true
			break
		}
	}
	if !found {
		return Task{}, ErrBadGitArgs
	}
	if dangerousGit(args) {
		return Task{}, fmt.Errorf("%w: destructive operations, command helpers and hooks need explicit approval", ErrBadGitArgs)
	}
	// --no-pager, terminal-only editors disabled in runner environment,
	// command-local hooks disabled even if the repository config enables them.
	argv := append([]string{"git", "-c", "core.hooksPath=/dev/null", "--no-pager"}, args...)
	return Task{Name: "git", Kind: "git", Description: "Git " + verb, Command: argv, TimeoutSeconds: 180}, nil
}

func vcsArgs(args []string) error {
	if len(args) < 1 || len(args) > 128 {
		return ErrBadGitArgs
	}
	for _, arg := range args {
		if arg == "" || len(arg) > 4096 || strings.ContainsAny(arg, "\x00\r\n") {
			return ErrBadGitArgs
		}
	}
	return nil
}

func dangerousGit(args []string) bool {
	verb := args[0]
	// Operations that execute an external helper or arbitrary process, alter
	// credentials/configuration, or remove local/remote history are excluded.
	switch verb {
	case "credential", "citool", "receive-pack", "upload-pack", "upload-archive",
		"send-email", "web--browse", "replace", "config":
		// git config can set external helpers/filters. Read-only config queries
		// remain available for diagnostics.
		if verb == "config" && len(args) > 1 {
			switch args[1] {
			case "--get", "--get-all", "--get-regexp", "--list", "-l", "--show-origin", "--show-scope":
				return false
			}
		}
		return true
	case "submodule":
		// submodule update and foreach can execute arbitrary commands.
		if len(args) > 1 && args[1] != "status" && args[1] != "summary" {
			return true
		}
	case "bisect":
		if len(args) > 1 && args[1] == "run" {
			return true
		}
	case "maintenance":
		return true
	case "stash":
		if len(args) > 1 && ghContains(args[1], "clear drop") {
			return true
		}
	case "tag":
		if len(args) > 1 && ghContains(args[1], "-d --delete") {
			return true
		}
	case "branch":
		if len(args) > 1 && ghContains(args[1], "-d -D --delete") {
			return true
		}
	case "clean":
		// Only previewing a clean is safe without a separate approval.
		if len(args) == 1 || !ghContains(args[1], "-n --dry-run") {
			return true
		}
	}
	for _, arg := range args[1:] {
		if verb == "push" && (strings.HasPrefix(arg, "+") ||
			strings.HasPrefix(arg, ":") || strings.Contains(arg, ":+")) {
			// +HEAD:branch forces a rewrite; :branch deletes the remote ref.
			return true
		}
		switch arg {
		case "-c", "-C", "-p", "-P", "--config-env", "--git-dir",
			"--work-tree", "--exec-path", "--namespace", "--upload-pack",
			"--receive-pack", "--exec", "--ext-diff", "--textconv",
			"--no-verify", "--force", "-f", "--force-with-lease",
			"-d",
			"--force-if-includes", "--mirror", "--delete", "-D",
			"--hard", "--discard-changes", "--allow-unrelated-histories",
			"--recurse-submodules", "--interactive", "--edit-todo":
			// -f is context-sensitive: force deletion/checkout/push should
			// require explicit approval; errors are preferred to unsafe writes.
			return true
		}
		for _, flag := range []string{
			"-c=", "--config-env=", "--git-dir=", "--work-tree=",
			"--exec-path=", "--namespace=", "--upload-pack=",
			"--receive-pack=", "--exec=", "--ext-diff=",
			"--textconv=", "--force=", "--force-with-lease=",
			"--recurse-submodules=", "--template=", "--strategy-option=",
			"--remote=", "--server-option=",
		} {
			if strings.HasPrefix(arg, flag) {
				return true
			}
		}
		if strings.HasPrefix(arg, "-C") && len(arg) > 2 {
			return true
		}
	}
	return false
}
