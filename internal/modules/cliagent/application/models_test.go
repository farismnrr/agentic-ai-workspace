package application_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

var errNoDocker = errors.New("no docker")

// printing is a sandbox whose program prints a fixed answer, and counts how often it was asked.
type printing struct {
	err     error
	said    string
	bundled string
	hosts   []string
	calls   int
}

func (p *printing) Run(_ context.Context, job application.Job, out io.Writer) (int, error) {
	p.calls++

	if slices.Contains(job.Argv, "--bundled") {
		_, _ = io.WriteString(out, p.bundled)

		return 0, nil
	}

	p.hosts = job.Hosts

	_, _ = io.WriteString(out, p.said)

	return 0, p.err
}

func newModelsService(box *printing, clock *time.Time) *application.Service {
	hosts := map[domain.Runtime][]string{domain.ChatGPT: {"chatgpt.com"}}

	return application.NewService(box, hosts, func() time.Time { return *clock })
}

const accountList = `{"models":[{"slug":"gpt-x","display_name":"GPT X","visibility":"list","priority":1,` +
	`"supported_reasoning_levels":[{"effort":"low"}]}]}`

func TestCodexModelsComeFromTheAccountAndAreKept(t *testing.T) {
	t.Parallel()

	now := time.Now()
	box := &printing{said: accountList}
	service := newModelsService(box, &now)

	for range 2 {
		models := service.Models(t.Context(), "u", domain.ChatGPT)
		if len(models) != 1 || models[0].ID != "gpt-x" {
			t.Fatalf("models = %+v", models)
		}
	}

	if box.calls != 1 || len(box.hosts) == 0 {
		t.Fatalf("asked %d times with hosts %v: want once, and with the program's hosts", box.calls, box.hosts)
	}
}

func TestCodexModelsFallBackToTheBuiltInListWhenTheAccountsCannotBeRead(t *testing.T) {
	t.Parallel()

	now := time.Now()
	box := &printing{err: errNoDocker, bundled: accountList}

	if models := newModelsService(box, &now).Models(t.Context(), "u", domain.ChatGPT); models[0].ID != "gpt-x" {
		t.Fatalf("models = %+v", models)
	}
}

func TestCodexModelsFallBackToTheProgramsOwnChoiceAndRetrySoon(t *testing.T) {
	t.Parallel()

	now := time.Now()
	box := &printing{err: errNoDocker}
	service := newModelsService(box, &now)

	models := service.Models(t.Context(), "u", domain.ChatGPT)
	if len(models) != 1 || models[0].ID != domain.DefaultModel {
		t.Fatalf("models = %+v", models)
	}

	service.Models(t.Context(), "u", domain.ChatGPT)

	if box.calls != 2 {
		t.Fatalf("asked %d times in a row", box.calls)
	}

	now = now.Add(2 * time.Minute)
	box.err, box.said = nil, accountList

	if models := service.Models(t.Context(), "u", domain.ChatGPT); models[0].ID != "gpt-x" {
		t.Fatalf("after a while, models = %+v", models)
	}
}

func TestClaudeModelsAreItsOwnNames(t *testing.T) {
	t.Parallel()

	now := time.Now()
	box := &printing{}

	if models := newModelsService(box, &now).Models(t.Context(), "u", domain.Claude); len(models) < 2 || box.calls != 0 {
		t.Fatalf("models = %+v, asked %d times", models, box.calls)
	}
}
