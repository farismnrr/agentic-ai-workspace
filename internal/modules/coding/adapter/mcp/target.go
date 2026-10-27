package mcp

import "strings"

// Target says which of the person's projects a call works in. Every tool that works in a project takes it as the
// optional argument project: a Project ID from list_projects. Left out, the call works in the app's own project.
type Target struct {
	Project string `json:"project,omitempty" jsonschema:"Project ID from list_projects; omit for your own project"`
	WorkID  string `json:"work_id,omitempty" jsonschema:"work session ID returned by begin_work"`
}

// WorkTarget requires a work ID for lifecycle operations while retaining the optional project selector.
type WorkTarget struct {
	Project string `json:"project,omitempty" jsonschema:"Project ID from list_projects; omit for your own project"`
	WorkID  string `json:"work_id" jsonschema:"work session ID returned by begin_work"`
}

func (t WorkTarget) wanted() string { return strings.TrimSpace(t.Project) }
func (t WorkTarget) workID() string { return strings.TrimSpace(t.WorkID) }

func (t Target) wanted() string { return strings.TrimSpace(t.Project) }

// targeted is an input that names a project.
type (
	targeted   interface{ wanted() string }
	workTarget interface{ workID() string }
)

func (t Target) workID() string { return strings.TrimSpace(t.WorkID) }

// projectOf is the project an input asks for, or "" for the app's own.
func projectOf(in any) string {
	if target, ok := in.(targeted); ok {
		return target.wanted()
	}

	return ""
}

func workIDOf(in any) string {
	if target, ok := in.(workTarget); ok {
		return target.workID()
	}
	return ""
}
