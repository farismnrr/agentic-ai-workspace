package application

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	toolRunCommand = "run_command"
	// maxLocalAddressRefusals is how many times a finish is sent back for naming an address only the sandbox reaches.
	maxLocalAddressRefusals = 2
)

// ServerStartPattern matches a shell command that starts a server or leaves a process running behind it. Such a
// process dies with the command's container and nobody outside the sandbox can reach it: an app is started with
// start_site, and only by the DevOps teammate. It is written to work in Go and in JavaScript, which checks the
// commands of a teammate's own program; the case does not matter.
const ServerStartPattern = `(\bnohup\b|\bsetsid\b|\bdisown\b|[^&]&\s*(;|$)|` +
	`\b(npm|pnpm|yarn|bun|npx|bunx)\s+(-{1,2}[\w-]+\s+)*(run\s+)?(-{1,2}[\w-]+\s+)*(dev|start|serve|preview)\b|` +
	`\b(nuxt|nuxi|next|vite|astro)\s+(dev|start|preview)\b|\bhttp\.server\b|\bhttp-server\b|\bflask\s+run\b|` +
	`\buvicorn\b|\bgunicorn\b|\bphp\s+-S\b|\bnode\s+\S*\.output/server)`

var (
	serverStarts = regexp.MustCompile(`(?i)` + ServerStartPattern)
	// leftRunning matches a command that detaches what it starts, so killing something else in it changes nothing.
	leftRunning = regexp.MustCompile(`(?i)\b(nohup|setsid|disown)\b`)
)

// StartsServer reports whether a shell command starts a server for the person to open, or leaves a process behind.
// A command that starts a server and kills it again leaves nothing behind, as a check of a built app does; one that
// detaches the server (nohup, setsid, disown) is never excused by a kill somewhere else in the same command.
func StartsServer(command string) bool {
	if !serverStarts.MatchString(command) {
		return false
	}

	return !strings.Contains(command, "kill") || leftRunning.MatchString(command)
}

// ServerHoldText says why a shell command may not run, and what to do instead. canStart is true for the teammate
// that has start_site.
func ServerHoldText(canStart bool) string {
	text := "Do not start a server with a shell command: it stops when the command ends, and the person cannot reach " +
		"it. Their browser opens the app from Sites. "
	if canStart {
		return text + "Start it with start_site (a free port; if one is taken, take the next)."
	}

	return text + "Only the DevOps teammate can start it, and it is not your job: finish your task and report " +
		"\"Needs the app started\"."
}

// localAddress matches an address that only the sandbox can reach.
var localAddress = regexp.MustCompile(`(?i)(127\.0\.0\.1|\blocalhost\b|\b0\.0\.0\.0\b)`)

// serverHold says why a shell command may not run: it starts a server the person could never open.
func (r *Runner) serverHold(call Call) string {
	if call.Name != toolRunCommand {
		return ""
	}

	var in struct {
		Command string `json:"command"`
	}

	if json.Unmarshal(call.Input, &in) != nil || !StartsServer(in.Command) {
		return ""
	}

	return ServerHoldText(r.role == nil || slices.Contains(r.role.Tools, toolStartSite))
}

// localAddressHeld sends back a finish that gives the person an address of the sandbox. The person opens a web
// app from Sites, from any computer; 127.0.0.1 and localhost mean nothing to them. Told a few times only.
func (r *Runner) localAddressHeld(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	if state.localAsked >= maxLocalAddressRefusals || !localAddress.MatchString(parseFinish(call.Input).Summary) {
		return false
	}

	state.localAsked++
	r.result(ctx, task, call.ID, "Your summary gives the person an address like 127.0.0.1 or localhost. They open the "+
		"app from a different computer, from Sites, and cannot use those. If the app is running, say it is in "+
		"Sites; if no one started it with start_site, have the DevOps teammate start it first. Then finish again "+
		"without any local address.", true)

	return true
}
