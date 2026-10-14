package domain

import (
	"errors"
	"strings"
)

// MaxCommandLength bounds one shell command line.
const MaxCommandLength = 4000

// ErrBadCommand means the command is empty or too long.
var ErrBadCommand = errors.New("give one command to run, at most 4000 characters")

// CommandTask turns a shell command line into a task. Unlike a named task it goes
// through bash, which is safe because the sandbox, not the command, is the boundary.
func CommandTask(command string, timeoutSeconds int) (Task, error) {
	command = strings.TrimSpace(command)
	if command == "" || len(command) > MaxCommandLength {
		return Task{}, ErrBadCommand
	}

	task := Task{
		Name: "command", Description: command, Command: []string{"bash", "-c", command},
		TimeoutSeconds: min(max(timeoutSeconds, 0), int(MaxTaskTimeout.Seconds())),
	}

	return task, nil
}
