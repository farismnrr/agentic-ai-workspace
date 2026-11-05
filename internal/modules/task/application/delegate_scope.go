package application

import (
	"encoding/json"
	"slices"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

var (
	writeToolNames = []string{
		"write_file", "edit_file", "delete_file", "move_file", "download_asset",
		"run_command", "run_task", "verify", "run_build", "start_site", "browser_test",
		"run_artifact", "marimo_execute", "molab_execute",
	}
	soloOnlyRoleKeys = []string{"planner", "devops"}
)

// CanParallel reports whether a role may run concurrently with other teammates in the same delegation batch.
// Only Developer, Researcher, QA, Security (and custom roles without solo-only
// duties like writing the PRD, asking the person, designing the global look, or starting the site) may run in parallel.
func CanParallel(role AgentRole) bool {
	if slices.Contains(soloOnlyRoleKeys, role.Key) {
		return false
	}

	if role.WritesPRD || slices.Contains(role.Tools, toolAskUser) ||
		slices.Contains(role.Tools, toolStartSite) || slices.Contains(role.Tools, "search_images") {
		return false
	}

	return true
}

func canParallel(role AgentRole) bool {
	return CanParallel(role)
}

func parallelRoleRefused(role AgentRole) string {
	if canParallel(role) {
		return ""
	}

	return "The " + role.Name + " must run on its own, not in parallel with other teammates. Call delegate for the " +
		role.Name + " in a separate turn."
}

// RoleCanWrite reports whether a role can write files or use an unbounded execution tool.
func RoleCanWrite(role AgentRole) bool {
	return slices.ContainsFunc(role.Tools, func(tool string) bool {
		return slices.Contains(writeToolNames, tool)
	})
}

func roleCanWrite(role AgentRole) bool {
	return RoleCanWrite(role)
}

// parallelScopeConflict checks whether a delegation may run in parallel and whether its write scope is disjoint.
func parallelScopeConflict(role AgentRole, scope []string, otherWriteCount int, active [][]string) string {
	if why := parallelRoleRefused(role); why != "" {
		return why
	}

	if !roleCanWrite(role) || (otherWriteCount == 0 && len(active) == 0) {
		return ""
	}

	if len(scope) == 0 {
		return "Parallel file-changing teammates each need a non-empty scope of file or folder prefixes " +
			"(for example [\"pages/invoices\"] and [\"pages/customers\"]). Give each a disjoint scope, or run " +
			"them one after the other."
	}

	for _, existing := range active {
		if len(existing) == 0 || scopesOverlap(scope, existing) {
			return "The scope [" + strings.Join(scope, ", ") + "] overlaps with another teammate working in " +
				"parallel. Give each parallel file-changing teammate a disjoint scope, or run overlapping work " +
				"one after the other."
		}
	}

	return ""
}

// scopeHold refuses a file-changing tool call when a scoped teammate tries to touch a path outside its scope.
func (r *Runner) scopeHold(call Call) string {
	if len(r.scope) == 0 {
		return ""
	}

	switch call.Name {
	case toolRunCommand, "run_task", "verify", "run_build", "start_site", "browser_test",
		"run_artifact", "marimo_execute", "molab_execute":
		return "This execution tool cannot be limited to your assigned file scope. Complete scoped file work " +
			"with the named file tools; ask the orchestrator to run unrestricted work in an exclusive turn."
	}

	// Git source mutations can change arbitrary paths (cherry-picks, merges,
	// commits, branch switches). Refuse them in parallel folder-scoped jobs;
	// do them in an exclusive delegation after the slices are complete.
	if call.Name == "git" || call.Name == "gh" {
		var in struct {
			Args []string `json:"args"`
		}
		if err := json.Unmarshal(call.Input, &in); err != nil ||
			!codingdomain.VCSReadOnly(call.Name, in.Args) {
			return "Git/GitHub mutations need an exclusive teammate turn, not a parallel file scope."
		}
		return ""
	}

	targets, checked := writeTargets(call)
	if !checked {
		if slices.Contains(writeToolNames, call.Name) {
			return "The file target could not be checked against your assigned scope. Use a named file tool " +
				"with a clear relative path."
		}
		return ""
	}

	for _, target := range targets {
		if !inScope(target, r.scope) {
			return "That file is outside your assigned scope (" + strings.Join(r.scope, ", ") + "). Do not change " +
				"files outside your scope; if a shared file needs changing, note it in your report to the orchestrator."
		}
	}

	return ""
}
