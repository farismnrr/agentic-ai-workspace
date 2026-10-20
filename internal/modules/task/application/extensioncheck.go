package application

import (
	"context"
	"path"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxExtensionRefusals is how many times a finish is held back for an extension nobody packaged.
const maxExtensionRefusals = 3

// isExtensionSource reports whether a file belongs to the source of a browser extension built with WXT, which is how
// the team builds one: its config, or a file under entrypoints/.
func isExtensionSource(file string) bool {
	clean := path.Clean(file)

	return strings.HasPrefix(path.Base(clean), "wxt.config.") || strings.HasPrefix(clean, "entrypoints/") ||
		strings.Contains(clean, "/entrypoints/")
}

// extensionNotPackaged reports whether extension source changed after the last time an extension was packaged.
func extensionNotPackaged(events []domain.Event) bool {
	var change, packaged int64

	for _, event := range events {
		switch {
		case event.Kind == domain.EventExtension:
			packaged = event.Seq
		case !event.Undone && anyExtensionSource(event.Files):
			change = event.Seq
		}
	}

	return change > packaged
}

func anyExtensionSource(files []string) bool {
	for _, file := range files {
		if isExtensionSource(file) {
			return true
		}
	}

	return false
}

// extensionHeld holds a finish back while a browser extension was written and not packaged: the person installs it
// from Extensions, which only lists what was deployed.
func (r *Runner) extensionHeld(ctx context.Context, task domain.Task, state *runState, call Call) bool {
	return r.holdFinish(ctx, task, call, &state.extensionAsked, maxExtensionRefusals, extensionNotPackaged,
		"The work includes a browser extension that is not packaged for the person yet. Have the teammate who deploys "+
			"(DevOps) build it and run deploy_extension with the folder that holds manifest.json, then finish. An "+
			"extension is installed from Extensions in the sidebar, not from Sites.")
}
