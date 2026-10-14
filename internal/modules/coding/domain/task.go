package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"
)

// Task limits.
const (
	DefaultTaskTimeout = 120 * time.Second
	MaxTaskTimeout     = 600 * time.Second
)

// Task errors.
var (
	ErrNoTasks     = errors.New("no tasks are set up; add .workspace/tasks.json to the workspace")
	ErrUnknownTask = errors.New("unknown task; use list_tasks to see the available names")
	ErrBadTasks    = errors.New("the tasks file is not valid")
	ErrTaskBusy    = errors.New("another task is still running; try again when it finishes")
)

var taskName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// Task is a named command the person has approved. Command is an argument
// vector run directly, never through a shell.
type Task struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

// Timeout is how long the task may run.
func (t Task) Timeout() time.Duration {
	if t.TimeoutSeconds <= 0 {
		return DefaultTaskTimeout
	}

	return min(time.Duration(t.TimeoutSeconds)*time.Second, MaxTaskTimeout)
}

// ParseTasks reads {"tasks": {"name": {...}}} and returns the tasks sorted by name.
func ParseTasks(data []byte) ([]Task, error) {
	var file struct {
		Tasks map[string]Task `json:"tasks"`
	}

	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadTasks, err)
	}

	tasks := make([]Task, 0, len(file.Tasks))

	for name, task := range file.Tasks {
		if !taskName.MatchString(name) || len(task.Command) == 0 || task.Command[0] == "" {
			return nil, fmt.Errorf("%w: task %q needs a lowercase name and a command", ErrBadTasks, name)
		}

		task.Name = name
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })

	return tasks, nil
}
