package bootstrap

import (
	"strings"
	"time"

	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func goTestsExecuted(command, output string, truncated bool) bool {
	if !taskapp.GoTestExecutionEvidenceAllowed(command) || truncated {
		return false
	}
	return hasTestPackageResult(output)
}

func hasTestPackageResult(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if successfulTestPackage(line) && !strings.Contains(line, "[no tests to run]") {
			return true
		}
	}
	return false
}

func successfulTestPackage(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 3 || fields[0] != "ok" {
		return false
	}
	if fields[2] == "(cached)" {
		return true
	}
	_, err := time.ParseDuration(fields[2])
	return err == nil
}
