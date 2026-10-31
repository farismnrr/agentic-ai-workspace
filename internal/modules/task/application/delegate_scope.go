package application

import (
	"encoding/json"
	"path"
	"slices"
	"strings"
)

var (
	writeToolNames   = []string{"write_file", "edit_file", "delete_file", "move_file", "download_asset"}
	soloOnlyRoleKeys = []string{"planner", "devops"}
)

// canParallel reports whether a role may run concurrently with other teammates in the same delegation batch.
// Only Developer, Researcher, QA, Security, Architect (and custom roles without solo-only
// duties like writing the PRD, asking the person, designing the global look, or starting the site) may run in parallel.
func canParallel(role AgentRole) bool {
	if slices.Contains(soloOnlyRoleKeys, role.Key) {
		return false
	}

	if role.WritesPRD || slices.Contains(role.Tools, toolAskUser) ||
		slices.Contains(role.Tools, toolStartSite) || slices.Contains(role.Tools, "search_images") {
		return false
	}

	return true
}

func parallelRoleRefused(role AgentRole) string {
	if canParallel(role) {
		return ""
	}

	return "The " + role.Name + " must run on its own, not in parallel with other teammates. Call delegate for the " +
		role.Name + " in a separate turn."
}

// roleCanWrite reports whether a role can create, edit, move or delete files in the project.
func roleCanWrite(role AgentRole) bool {
	return slices.ContainsFunc(role.Tools, func(tool string) bool {
		return slices.Contains(writeToolNames, tool)
	})
}

// cleanScope normalizes relative file or folder prefixes and drops root or parent-traversal entries.
func cleanScope(raw []string) []string {
	var out []string

	for _, item := range raw {
		cleaned, ok := cleanScopePath(item)
		if ok && !slices.Contains(out, cleaned) {
			out = append(out, cleaned)
		}
	}

	return out
}

func cleanScopePath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return "", false
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}

	return cleaned, true
}

// pathsOverlap reports whether two normalized paths name the same file/folder or one is inside the other.
func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// scopesOverlap reports whether any path prefix in a overlaps with any path prefix in b.
func scopesOverlap(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if pathsOverlap(left, right) {
				return true
			}
		}
	}

	return false
}

// inScope reports whether target lies inside at least one prefix of scope.
func inScope(target string, scope []string) bool {
	cleaned, ok := cleanScopePath(target)
	if !ok {
		return false
	}

	for _, prefix := range scope {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return true
		}
	}

	return false
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

	targets, checked := writeTargets(call)
	if !checked {
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

func writeTargets(call Call) ([]string, bool) {
	switch call.Name {
	case "write_file", "edit_file", "delete_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return nil, false
		}
		return []string{args.Path}, true
	case "move_file":
		var args struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return nil, false
		}
		return []string{args.From, args.To}, true
	default:
		return nil, false
	}
}
