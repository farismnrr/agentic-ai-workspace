package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"sync"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// siteToolNames lists the project capabilities mediated by the team MCP server.
// The vendor's native file tools have different names (Read/Glob/Write, etc.)
// and can be unavailable in restricted sessions. Expose the existing bounded
// project-file tools here too, so enabled read/write capabilities are always
// discoverable by their app names, with the same role, owner-switch, risk,
// project-scope and undo checks as normal project tool calls.
// Site, browser, CI, web, and Git operations remain server-mediated.
var siteToolNames = []string{
	"list_files", "find_files", "read_file", "search_code", "list_tasks", "view_image",
	"edit_file", "write_file", "delete_file", "move_file",
	toolStartSite, "stop_site", "deploy_extension", "web_search", "fetch_page", "search_images", "download_asset",
	"screenshot", "screenshot_session", toolRequestEgress, "ci_logs", "docker_host_status",
	"run_task", "verify", "browser_test", "read_browser_screenshot", "run_build", "run_artifact", "git", "gh",
	"marimo_execute", "molab_execute", "ask_user",
	// DevOps may use this only when the server's role gate AND the owner tool
	// switch authorize it. It remains a medium-risk approval-gated sandbox
	// action; Developer's role intentionally never receives run_command.
	"run_command",
}

// SiteToolNames returns a copy of the mediated site and project tool names.
func SiteToolNames() []string {
	return append([]string(nil), siteToolNames...)
}

// SiteSession serves site, Git and GitHub tools to a teammate's program. Each call goes through the same code as a call of the
// app's own loop (the Security check, the person's approval, the link in the thread).
type SiteSession struct {
	inline  *Runner
	state   *runState
	results *capturedResults
	stopped *outcome
	jobID   string
	tools   []CLITool
	task    domain.Task
	role    AgentRole
	mu      sync.Mutex
}

type siteSession = SiteSession

// NewSiteSessionForTest creates a minimal SiteSession for testing tool bridge checks.
func NewSiteSessionForTest(tools []CLITool) *SiteSession {
	return &SiteSession{tools: tools}
}

// BridgedTools uses the same role-filtered, owner-switch-filtered names
// for actual MCP registration as the prompt advertises.
func BridgedTools(defs []domain.ToolDef, enabled []string) []CLITool {
	return bridgedTools(defs, enabled)
}

// bridgedTools uses the same role-filtered, owner-switch-filtered names
// for actual MCP registration as the prompt advertises. No tool may be
// exposed only because it exists in the global server catalogue.
func bridgedTools(defs []domain.ToolDef, enabled []string) []CLITool {
	allowed := roleToolSet(enabled)
	tools := []CLITool{}
	for _, def := range defs {
		if contains(siteToolNames, def.Name) && allowed[def.Name] {
			tools = append(tools, toolOfDef(def))
		}
	}
	return tools
}

// siteBridge exposes bounded project-file, site and VCS tools to the vendor,
// always enforcing the server-owned role and owner's live tool switches.
func (r *Runner) siteBridge(task domain.Task, role AgentRole, jobID string) *siteSession {
	tools := bridgedTools(r.deps.Tools.Definitions(), role.Tools)

	if len(tools) == 0 {
		return nil
	}

	results := &capturedResults{answers: map[string]capturedResult{}}
	inline := *r
	inline.deps.Transcript = results

	return &siteSession{
		inline: &inline, results: results, tools: tools, task: task, jobID: jobID,
		state: &runState{budget: &budget{left: r.lim.delegateSteps()}, segment: r.lim.delegateSteps()}, role: role,
	}
}

func (s *siteSession) bridge() *CLIBridge {
	if s == nil {
		return nil
	}

	return &CLIBridge{Tools: s.tools, Call: s.call}
}

// call answers one call of the program like the loop answers one of the model's. The call keeps its name between
// turns, so a step the person already approved is not asked about again when the job starts over.
func (s *siteSession) call(ctx context.Context, name string, input json.RawMessage) CLIToolResult {
	// A client must never invoke a capability that was omitted from its
	// registered MCP inventory. Re-check the exact session list, not only the
	// global bridge catalogue, even if it sends a crafted tool-call request.
	visible := false
	for _, tool := range s.tools {
		if tool.Name == name {
			visible = true
			break
		}
	}
	if !contains(siteToolNames, name) || !visible {
		return CLIToolResult{Text: "There is no enabled team tool called " + name + " for this role.", Failed: true}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped != nil {
		return CLIToolResult{Text: "Stop now: the job is over for this turn.", Failed: true}
	}

	sum := sha256.Sum256(append([]byte(s.jobID+"\x00"+name+"\x00"), input...))
	id := "site-" + hex.EncodeToString(sum[:8])

	out, done := s.inline.handle(ctx, s.task, s.state, Call{ID: id, Name: name, Input: input})
	if done {
		s.stopped = &out

		return CLIToolResult{Text: stopText(out), Failed: out.stop == stopFailed}
	}

	got := s.results.take(id)

	result := CLIToolResult{Text: got.text, Failed: got.failed}
	if !got.failed && s.role.Key == "qa" && (name == "browser_test" || name == "read_browser_screenshot") && got.image != "" && s.inline.deps.Shots != nil {
		data, err := s.inline.deps.Shots.Load(ctx, s.task.ID, got.image)
		if err == nil && len(data) > 0 && len(data) <= 5<<20 {
			if config, decodeErr := png.DecodeConfig(bytes.NewReader(data)); decodeErr == nil && config.Width <= 4096 && config.Height <= 4096 {
				result.Media = []CLIMedia{{MIMEType: "image/png", Data: data}}
			}
		}
	}

	return result
}

// Call answers one call of the program from an external caller or test.
func (s *SiteSession) Call(ctx context.Context, name string, input json.RawMessage) CLIToolResult {
	return s.call(ctx, name, input)
}

// stoppedOutcome is the outcome that ends the job early, when a call asked for it.
func (s *siteSession) stoppedOutcome() (outcome, bool) {
	if s == nil || s.stopped == nil {
		return outcome{}, false
	}

	return *s.stopped, true
}
