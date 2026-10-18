package application

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	toolReadReference = "read_reference"
	// maxSummary keeps one line of a list short.
	maxSummary = 140
)

// appendLibrary lists what a teammate can look up, so it knows the names to ask for.
func appendLibrary(out *strings.Builder, refs []Reference) {
	if len(refs) == 0 {
		return
	}

	out.WriteString("\nYour library. Read an item with read_reference, using its full name, when the job needs it " +
		"(a style family before you design, a template before a page, a craft rule before you check):\n")

	for _, ref := range refs {
		out.WriteString("- " + ref.Name + ": " + clipTo(ref.Summary, maxSummary) + "\n")
	}
}

// hasLibrary reports whether this runner is a teammate with a library to read.
func (r *Runner) hasLibrary() bool {
	return r.role != nil && len(r.role.Library) > 0 && r.role.Lookup != nil
}

func (r *Runner) lookup(name string) (string, bool) {
	if r.role == nil || r.role.Lookup == nil {
		return "", false
	}

	return r.role.Lookup(name)
}

// readReferenceTool is the loop's read_reference tool, for a teammate with a library.
func readReferenceTool() domain.ToolDef {
	return domain.ToolDef{
		Name:        toolReadReference,
		Description: "Read one item of your library by its full name, such as style/minimal or craft/color.",
		Schema:      json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`),
	}
}

// readReference answers with the text of the library item the teammate asked for.
func (r *Runner) readReference(ctx context.Context, task domain.Task, call Call) {
	var args struct {
		Name string `json:"name"`
	}

	_ = json.Unmarshal(call.Input, &args)

	if text, ok := r.lookup(strings.TrimSpace(args.Name)); ok {
		r.resultUpTo(ctx, task, call.ID, text, false, maxSkillResult)

		return
	}

	r.result(ctx, task, call.ID, "There is no library item with that name. Use a full name from your library.", true)
}
