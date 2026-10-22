package application

// ToolScope is resolved from the running task and never comes from model input.
type ToolScope struct {
	UserID    string
	ProjectID string
	TaskID    string
}
