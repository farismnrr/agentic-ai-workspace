package clistream

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"

func changeStep(tool string, in claudeInput, failed bool) application.CLIEvent {
	path := relative(in.FilePath + in.NotebookRef)
	event := application.CLIEvent{Kind: application.CLIChange, Technical: tool + " " + path}

	if failed {
		event.Text = "Tried to change " + path + ", but it did not work"

		return event
	}

	event.Files = []string{path}
	event.Text = "Changed " + path

	if tool == "Write" {
		event.Text = "Wrote " + path
	}

	return event
}

func commandStep(in claudeInput, failed bool) application.CLIEvent {
	event := application.CLIEvent{Kind: application.CLICommand, Technical: in.Command, Text: in.Description}
	if event.Text == "" {
		event.Text = "Ran " + brief(in.Command, commandChars)
	}

	if failed {
		event.ExitCode = 1
	}

	return event
}
