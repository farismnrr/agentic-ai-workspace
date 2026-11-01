package connectors_test

import (
	"context"
	"testing"

	codingapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type qaVerifySpy struct {
	ran      bool
	readonly bool
	gitOnly  bool
	args     []string
}

func (s *qaVerifySpy) Run(ctx context.Context, task domain.Task) (codingapp.RunResult, error) {
	s.ran = true
	s.readonly = domain.ReadOnlyWorkspace(ctx)
	s.gitOnly = domain.GitOnly(ctx)
	s.args = append([]string(nil), task.Command...)
	return codingapp.RunResult{ExitCode: 0}, nil
}

func TestQAWithoutTaskFileCanVerifySourceReadOnly(t *testing.T) {
	spy := &qaVerifySpy{}
	// nil task file is intentional: QA verification does not consult it.
	tasks := codingapp.NewTasks(nil, spy)
	for _, args := range [][]string{
		{"go", "test", "./test/go/app/connectors", "-count=1"},
		{"go", "vet", "./internal/modules/coding/..."},
		{"npm", "run", "check"},
		{"pnpm", "run", "lint"},
		{"python3", "-m", "pytest", "-q"},
		{"cargo", "test"},
		{"make", "verify"},
		{"dotnet", "test"},
		{"npx", "--no-install", "playwright", "test"},
		{"uv", "run", "pytest", "-q"},
		{"gofmt", "-l", "internal/modules/coding/domain"},
	} {
		spy.ran = false
		result, err := tasks.Verify(context.Background(), args)
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("QA verification unexpectedly refused %q: %v", args, err)
		}
		if !spy.ran || !spy.readonly || spy.gitOnly {
			t.Fatalf("QA verification escaped readonly/secretless executor: %+v", spy)
		}
	}
}

func TestQAVerifyRejectsArbitraryShellAndDestructiveOptions(t *testing.T) {
	for _, args := range [][]string{
		{}, {"bash", "-lc", "go test ./..."},
		{"sh", "-c", "pytest"}, {"python", "-c", "print(1)"},
		{"go", "generate", "./..."}, {"go", "test", "./...", "-exec", "/tmp/evil"},
		{"npm", "run", "deploy"}, {"npm", "run", "build"},
		{"npm", "run", "test", "--", "--fix"},
		{"cargo", "build"}, {"python3", "-m", "pip", "install", "x"},
	} {
		if _, err := domain.VerifyTask(args); err == nil {
			t.Errorf("unsafe verification accepted: %q", args)
		}
	}
}
