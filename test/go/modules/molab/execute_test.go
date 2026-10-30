package molab_test

import (
	"context"
	"strings"
	"testing"

	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	molab "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
)

type fakeRunner struct {
	calls  []codingdomain.Task
	hosts  []string
	tokens []string
	code   int
	output string
}

func (r *fakeRunner) RunMoLab(_ context.Context, task codingdomain.Task, token, host string) (codingapp.RunResult, error) {
	r.calls = append(r.calls, task)
	r.hosts = append(r.hosts, host)
	r.tokens = append(r.tokens, token)
	return codingapp.RunResult{ExitCode: r.code, Output: r.output}, nil
}

func TestMCPExecutionInspectsBeforeRunningAndKeepsSecretOutOfArgv(t *testing.T) {
	r := &fakeRunner{output: "ready"}
	secret := "SYNTHETIC-CODE-123"
	result := molab.Execute(context.Background(), r, "https://sb-demo.sb.molab.run/", secret,
		"import marimo as mo; mo.status.toast('ready')")
	if result.Problem != "" || len(r.calls) != 2 {
		t.Fatalf("two executions expected: %#v calls=%d", result, len(r.calls))
	}
	first := r.calls[0].Command
	if !strings.Contains(first[len(first)-1], "help(cm)") {
		t.Fatal("first call did not inspect code mode")
	}
	if strings.Contains(first[len(first)-1], "status.toast") {
		t.Fatal("user code was run before inspection")
	}
	if !strings.Contains(r.calls[1].Command[len(r.calls[1].Command)-1], "status.toast") {
		t.Fatal("second call did not run user code")
	}
	for i, call := range r.calls {
		if r.hosts[i] != "sb-demo.sb.molab.run" || r.tokens[i] != secret {
			t.Fatal("incorrect exact host or token scope")
		}
		if strings.Contains(strings.Join(call.Command, " "), secret) {
			t.Fatal("credential leaked into command arguments")
		}
	}
}

func TestMCPExecutionRefusesInvalidOriginAndRedactsServerEcho(t *testing.T) {
	r := &fakeRunner{output: "SYNTHETIC-CODE-123", code: 1}
	result := molab.Execute(context.Background(), r, "https://sb-demo.sb.molab.run/", "SYNTHETIC-CODE-123", "1+1")
	if strings.Contains(result.Problem, "SYNTHETIC-CODE-123") || len(r.calls) != 1 {
		t.Fatal("failed inspection ran user code or leaked secret")
	}
	r.calls = nil
	invalid := molab.Execute(context.Background(), r, "https://localhost/", "SYNTHETIC-CODE-123", "1+1")
	if invalid.Problem == "" || len(r.calls) != 0 {
		t.Fatal("invalid host reached sandbox")
	}
}
