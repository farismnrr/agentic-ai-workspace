package application

import (
	"slices"
	"strings"
)

// preActionToolSOP is generated from the actual enabled role tools, not a
// static capabilities claim. It is sent at the start of a vendor teammate's
// turn, before the assignment, so tool selection is deterministic rather
// than trial-and-error. The server bridge uses the same filtered list.
func preActionToolSOP(role string, enabled []string) string {
	tools := slices.Clone(enabled)
	slices.Sort(tools)

	var out strings.Builder
	out.WriteString("\nMANDATORY PRE-ACTION TOOL SOP (read before the first tool call):\n")
	if len(tools) == 0 {
		out.WriteString("- No server project tools are enabled for this role in this session. Do not guess or use an unauthorized shell fallback; report the missing permission to the orchestrator.\n")
	} else {
		out.WriteString("- This role's ENABLED project tools: " + strings.Join(tools, ", ") + ". " +
			"Use only actually exposed tools; a missing tool cannot be replaced by shell commands.\n")
	}
	out.WriteString("- First choose the matching allowed server tool, check prerequisites and repository rules, then act once. " +
		"If a tool is absent, disabled or errors, report the precise blocker instead of trying random transports or proxies.\n")

	// The vendor also has native file commands and the app has external
	// work-lifecycle tools. Advertise the *actual* MCP team bridge subset so
	// an agent does not mistake those capabilities for available server tools.
	var bridged []string
	for _, name := range tools {
		if slices.Contains(siteToolNames, name) {
			bridged = append(bridged, name)
		}
	}
	if len(bridged) > 0 {
		out.WriteString("- ACTUAL TEAM MCP TOOLS: " + strings.Join(bridged, ", ") + ". " +
			"Use the server tool with that exact name; vendor built-ins are not a substitute for a missing project permission.\n")
	} else {
		out.WriteString("- No team MCP project tools were exposed to this role. Report the permission or tool-switch blocker.\n")
	}
	if slices.Contains(bridged, "read_file") {
		out.WriteString("- Project source inspection: prefer team MCP read_file with {\"path\":\"relative/file.ext\"}, " +
			"find_files for paths, search_code for content, and list_files for a folder, only when listed above. " +
			"Do not report that file inspection is unavailable without checking the named team tools.\n")
	} else {
		out.WriteString("- Project source inspection: team read_file is NOT available. " +
			"Ask the owner to enable the read_file tool in Settings > Spaces & tools, or report the missing role permission. " +
			"Do not use shell or a different project as a bypass.\n")
	}
	if role == "developer" {
		if !slices.Contains(bridged, "write_file") && !slices.Contains(bridged, "edit_file") {
			out.WriteString("- Editing is unavailable to this Developer session. Check edit_file/write_file switches; do not promise changes.\n")
		}
		out.WriteString("- Developer intentionally has NO unrestricted run_command shell. Use authorized read_file/edit_file/write_file and server git/gh. " +
			"Delegate named builds, application start/recreate and command diagnostics to DevOps; do not claim a missing shell is a disconnected tool.\n")
	}
	if role == "devops" {
		if slices.Contains(bridged, "run_command") {
			out.WriteString("- DevOps has the server run_command tool when needed for bounded diagnostics, with existing risk approval and sandbox limits. " +
				"Prefer an owner-configured run_build for builds and start_site for app lifecycle; never run shell Git/gh or Docker daemon commands.\n")
		} else {
			out.WriteString("- DevOps server run_command is NOT enabled in this session. Check the owner switch and role. " +
				"Use run_build/start_site when enabled; do not fall back to vendor Bash or bypass the proxy.\n")
		}
	}
	out.WriteString("- External Coding lifecycle names such as get_work_state, report_progress and complete_work " +
		"are not teammate MCP tools. Finish an internal delegation with your final report; the external Orchestrator calls complete_task.\n")

	if slices.Contains(tools, "git") || slices.Contains(tools, "gh") {
		out.WriteString("- Git / GitHub: use the team MCP tool named git (args such as [\"status\", \"--short\"] or " +
			"[\"push\", \"origin\", \"dev\"]) and/or gh (args such as [\"auth\", \"status\"]) ONLY if listed above. " +
			"Verify checkout, expected HEAD, clean state, remote and fast-forward before a non-force push. " +
			"Restricted vendor CLI shell has no owner GitHub token/egress; NEVER use shell git/gh, curl, alternate proxies or DNS bypass. " +
			"CONNECT_GITHUB_REQUIRED means the owner must connect GitHub under Settings.\n")
	} else {
		out.WriteString("- Git / GitHub: git/gh server tools are NOT enabled for this role. " +
			"Delegate to an authorized Developer, DevOps or Researcher as appropriate; never try shell git/gh.\n")
	}

	if role == "qa" && slices.Contains(tools, "browser_test") {
		out.WriteString("- Browser / Playwright E2E: before calling browser_test, confirm DevOps has started the project Site successfully through start_site. " +
			"If no Site is running, that is a missing prerequisite, NOT proof the browser is broken. Ask DevOps for the valid app start command/port and wait for a successful Site response. " +
			"Use browser_test actions [{\"action\":\"goto\",\"path\":\"\"}] for root, NOT '/' (paths are relative). " +
			"Then call read_browser_screenshot with the returned screenshot_ref when evidence is needed. " +
			"Never launch shell Playwright, arbitrary browser scripts or a different host as a fallback.\n")
	} else {
		out.WriteString("- Browser / Playwright E2E: browser_test and read_browser_screenshot belong to enabled QA. " +
			"Delegate browser actions to QA; screenshots alone do not replace an interactive E2E check. " +
			"If QA browser tools are disabled, report that switch instead of running shell Playwright.\n")
	}

	if slices.Contains(tools, "ci_logs") {
		out.WriteString("- CI: retrieve failed job logs using the server ci_logs tool, not gh run view --log-failed.\n")
	} else {
		out.WriteString("- CI: ask an authorized DevOps worker to retrieve logs through ci_logs.\n")
	}
	if slices.Contains(tools, "start_site") {
		out.WriteString("- Site preview: start_site/stop_site are the authorized route; never start a persistent web server through Bash.\n")
	}
	out.WriteString("- One issue stays in one task. Preserve unrelated/dirty workspace files; no force-push, cleanup or permission bypass without explicit owner authorization.\n")
	return out.String()
}

// preActionLeadSOP is sent before team planning/delegation: the lead never
// performs privileged work directly and must assign the right owner first.
const preActionLeadSOP = "\nMANDATORY PRE-ACTION TOOL SOP for the orchestrator: " +
	"Before delegating, decide which role owns each operation and name the exact server tool in the brief. " +
	"Git/git push/gh goes to Developer or DevOps (Researcher only where allowed) through the server git/gh tool, " +
	"not the teammate's shell. Browser/Playwright E2E goes to QA through browser_test; " +
	"start_site must already be running; browser_test goto path is relative, empty for root (not '/'); " +
	"read_browser_screenshot retrieves screenshot_ref evidence. " +
	"CI failures go to DevOps through ci_logs, not shell gh log downloads. " +
	"For QA browser testing, first assign DevOps to build the app if needed and start_site using a valid command/port; " +
	"only after a successful Site start should QA invoke browser_test. Site-not-running is a prerequisite, not a browser failure. " +
	"Developer is intentionally denied run_command; DevOps uses run_build/start_site and may use server run_command only with the owner's enabled switch and existing approvals. " +
	"Internal teammates finish by final report; the external lead calls complete_task, not team complete_work. " +
	"If a role's tool is missing/disabled, record the exact blocker and request that permission; " +
	"never ask the worker to bypass proxy, run shell Git, use shell Playwright or silently change roles. " +
	"Do not create extra tasks per delegation.\n"


// soloPreActionSOP guides the app's own model when no limited vendor
// teammate is involved. Solo runners can have separately authorized
// credentials, so this does not falsely assert they share the CLI sandbox.
const soloPreActionSOP = "\nPRE-ACTION TOOL SOP: Before any Git, browser, CI or site work, inspect the tools actually available and the owner's switches. " +
	"Prefer the built-in git/gh tools for GitHub operations when offered, and check remote, branch, current HEAD, clean state and fast-forward before push. " +
	"Never bypass HTTPS_PROXY, reveal a token or retry an unavailable authorized tool using a different transport. " +
	"Browser/Playwright E2E belongs to QA using browser_test goto path=\"\" for Site root (never '/'), " +
	"then read_browser_screenshot on a running Site; " +
	"start_site starts the preview (never a background shell), and screenshot alone is not interactive E2E. " +
	"Use ci_logs for failed CI logs rather than guessing from gh downloads. " +
	"If the required tool or permission is unavailable, report that blocker rather than improvising.\n"
