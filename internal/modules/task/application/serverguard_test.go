package application_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func devops() application.AgentRole {
	return application.AgentRole{
		Key: "devops", Name: "DevOps", Purpose: "Runs the app.",
		Tools: []string{"run_command", "start_site", "stop_site"},
	}
}

func TestAShellCommandThatStartsAServerIsRefused(t *testing.T) {
	t.Parallel()

	for _, command := range []string{
		"npm run dev", "cd app && bun --bun run dev &", "nohup node server.js &", "python3 -m http.server 8000",
		"nuxt dev", "yarn start", "uvicorn app:app", "node .output/server/index.mjs",
	} {
		r := newTeamRig(nil, []application.AgentRole{devops()},
			delegate("d1", "devops", "Run it"),
			call("c1", "run_command", map[string]string{"command": command}),
			report("r1", "Done"), finish("Done"),
		)
		r.run(r.startTeam(t, "Run the app").ID)

		if len(r.tools.executed) != 0 {
			t.Errorf("%q must not run, got %v", command, r.tools.executed)
		}
	}
}

func TestOrdinaryCommandsStillRun(t *testing.T) {
	t.Parallel()

	for _, command := range []string{"bun test", "npm run build && npm test", "go build ./... && echo ok", "ls | wc -l"} {
		r := newTeamRig(nil, []application.AgentRole{devops()},
			delegate("d1", "devops", "Check it"),
			call("c1", "run_command", map[string]string{"command": command}),
			report("r1", "Done"), finish("Done"),
		)
		r.tools.risks["run_command"] = domain.RiskNone
		r.run(r.startTeam(t, "Check").ID)

		if len(r.tools.executed) != 1 {
			t.Errorf("%q must run, got %v", command, r.tools.executed)
		}
	}
}

func TestAFinishThatGivesALocalAddressIsSentBack(t *testing.T) {
	t.Parallel()

	r := newRig(nil,
		finish("The app runs at http://127.0.0.1:3000"),
		finish("It is in Sites."),
	)
	task := r.start(t, "Say where it is")
	r.run(task.ID)

	var told bool

	for _, req := range r.model.requests {
		for _, m := range req.Messages {
			for _, b := range m.Blocks {
				told = told || strings.Contains(b.Text, "127.0.0.1 or localhost")
			}
		}
	}

	if !told || r.task(t, task.ID).Summary != "It is in Sites." {
		t.Fatalf("told=%v summary=%q", told, r.task(t, task.ID).Summary)
	}
}

func TestACheckThatKillsItsOwnServerIsNotRefused(t *testing.T) {
	t.Parallel()

	r := newTeamRig(nil, []application.AgentRole{devops()},
		delegate("d1", "devops", "Check it"),
		call("c1", "run_command", map[string]string{
			"command": "python3 -m http.server 4999 & sleep 2; curl -s :4999; kill $!",
		}),
		report("r1", "Done"), finish("Done"),
	)
	r.tools.risks["run_command"] = domain.RiskNone
	r.run(r.startTeam(t, "Check").ID)

	if len(r.tools.executed) != 1 {
		t.Fatalf("a self-cleaning check must run, got %v", r.tools.executed)
	}
}

func TestAFileThatLinksAFontWebsiteIsRefusedAndOneThatDoesNotIsWritten(t *testing.T) {
	t.Parallel()

	link := `<link href="https://fonts.googleapis.com/css2?family=Inter" rel="stylesheet">`

	for content, want := range map[string]int{link: 0, `<h1 style="font-family:Inter">Hi</h1>`: 1} {
		r := newTeamRig(nil, []application.AgentRole{developer()},
			delegate("d1", "developer", "Write the page"),
			call("c1", "edit_file", map[string]string{"path": "index.html", "new_text": content}),
			report("r1", "Done"), finish("Done"),
		)
		r.tools.risks["edit_file"] = domain.RiskNone
		r.run(r.startTeam(t, "Page").ID)

		if len(r.tools.executed) != want {
			t.Errorf("content %q: executed %v, want %d", content, r.tools.executed, want)
		}
	}
}
