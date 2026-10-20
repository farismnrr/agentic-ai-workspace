package mcp

import "strings"

// Target says which of the person's projects a call works in. Every tool that works in a project takes it as the
// optional argument project: a Project ID from list_projects. Left out, the call works in the app's own project.
type Target struct {
	Project string `json:"project,omitempty" jsonschema:"Project ID from list_projects; omit for your own project"`
}

func (t Target) wanted() string { return strings.TrimSpace(t.Project) }

// targeted is an input that names a project.
type targeted interface{ wanted() string }

// projectOf is the project an input asks for, or "" for the app's own.
func projectOf(in any) string {
	if target, ok := in.(targeted); ok {
		return target.wanted()
	}

	return ""
}
