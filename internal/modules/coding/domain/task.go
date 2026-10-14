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
	Name           string   `json:"name,omitempty"`
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

// Limits on what a checks file may hold.
const (
	MaxTasks       = 12
	MaxCommandArgs = 24
	MaxArgLength   = 300
)

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
		task.Name = name
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Name < tasks[j].Name })

	if err := ValidateTasks(tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// ValidateTasks checks names, commands and timeouts before they are saved or run.
func ValidateTasks(tasks []Task) error {
	if len(tasks) > MaxTasks {
		return fmt.Errorf("%w: at most %d checks", ErrBadTasks, MaxTasks)
	}

	seen := map[string]bool{}

	for _, task := range tasks {
		if err := validateTask(task); err != nil {
			return err
		}

		if seen[task.Name] {
			return fmt.Errorf("%w: %q appears twice", ErrBadTasks, task.Name)
		}

		seen[task.Name] = true
	}

	return nil
}

func validateTask(task Task) error {
	if !taskName.MatchString(task.Name) {
		return fmt.Errorf("%w: %q needs a lowercase name such as test or lint", ErrBadTasks, task.Name)
	}

	if len(task.Command) == 0 || task.Command[0] == "" || len(task.Command) > MaxCommandArgs {
		return fmt.Errorf("%w: %q needs a command", ErrBadTasks, task.Name)
	}

	for _, arg := range task.Command {
		if len(arg) > MaxArgLength {
			return fmt.Errorf("%w: a word in %q is too long", ErrBadTasks, task.Name)
		}
	}

	if task.TimeoutSeconds < 0 || time.Duration(task.TimeoutSeconds)*time.Second > MaxTaskTimeout {
		return fmt.Errorf("%w: %q may run for at most %d seconds", ErrBadTasks, task.Name, int(MaxTaskTimeout.Seconds()))
	}

	return nil
}

// EncodeTasks writes tasks in the file format ParseTasks reads.
func EncodeTasks(tasks []Task) ([]byte, error) {
	file := struct {
		Tasks map[string]Task `json:"tasks"`
	}{Tasks: make(map[string]Task, len(tasks))}

	for _, task := range tasks {
		name := task.Name
		task.Name = ""
		file.Tasks[name] = task
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode tasks: %w", err)
	}

	return data, nil
}
