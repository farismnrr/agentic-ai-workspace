package application

// ToolScope is resolved from the running task and never comes from model input.
type ToolScope struct {
	UserID    string
	ProjectID string
	TaskID    string
	RoleKey   string // server-assigned teammate role; empty for the solo run
}
