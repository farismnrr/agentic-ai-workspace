package application

import (
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func meaningfulCheck(run domain.RunInfo) bool {
	return knownCheckCommand(run.Command) && !noOpCheckCommand(run.Command) &&
		(!isGoTestCommand(run.Command) || GoTestExecutionEvidenceAllowed(run.Command) && run.TestExecuted)
}

func isGoTestCommand(command string) bool {
	fields := strings.Fields(command)
	return len(fields) >= 2 && fields[0] == "go" && fields[1] == "test"
}

func noOpCheckCommand(command string) bool {
	words, ok := literalCommandWords(command)
	if !ok {
		return true
	}
	for index, field := range words {
		if noOpFlag(field) || field == "-exec" || strings.HasPrefix(field, "-exec=") ||
			(field == "-count" || field == "-test.count") && index+1 < len(words) && words[index+1] == "0" {
			return true
		}
	}
	return false
}

func noOpFlag(field string) bool {
	return field == "-list" || strings.HasPrefix(field, "-list=") || field == "-n" || field == "-c" ||
		field == "-count=0" || field == "-test.count=0" || field == "-test.list" ||
		strings.HasPrefix(field, "-test.list=")
}
