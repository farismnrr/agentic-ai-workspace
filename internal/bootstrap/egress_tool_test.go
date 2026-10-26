package bootstrap

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func askFor(tools *agentTools, task, input string) taskapp.ToolResult {
	return tools.Execute(context.Background(), taskapp.ToolScope{UserID: "u1", ProjectID: "p1", TaskID: task},
		taskapp.Call{ID: "c", Name: "request_egress", Input: json.RawMessage(input)})
}

func hasTool(defs []taskdomain.ToolDef, name string) bool {
	for _, d := range defs {
		if d.Name == name {
			return true
		}
	}

	return false
}

func rows(t *testing.T, rt *egressRuntime, where string) int {
	t.Helper()

	var n int

	err := rt.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM egress_grants `+where).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}

	return n
}

func TestTheToolDoesNotExistUnlessGrantsAreRunning(t *testing.T) {
	off := &agentTools{}

	if hasTool(off.Definitions(), "request_egress") {
		t.Fatal("the model must not be told about a tool that is off")
	}

	if res := askFor(off, "t1", `{"host":"a.example.com","reason":"x"}`); !res.IsError {
		t.Fatalf("calling it anyway must fail: %+v", res)
	}
}

func TestAnAgentCanOnlyAskAndTheScopeIsTheServers(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	rt := startEgress(t, db, egressHub(t), dir)
	tools := &agentTools{egress: rt}

	if !hasTool(tools.Definitions(), "request_egress") {
		t.Fatal("the tool must be offered when grants run")
	}

	forged := `{"host":"Marimo.Example.com","reason":"open the notebook",` +
		`"project_id":"p2","task_id":"t2","status":"approved"}`

	res := askFor(tools, "t1", forged)
	if res.IsError || !strings.Contains(res.Output, "cannot approve") {
		t.Fatalf("%+v", res)
	}

	exact := "WHERE task_id = 't1' AND project_id = 'p1' AND status = 'pending' AND host = 'marimo.example.com' " +
		"AND requested_by_kind = 'agent' AND decided_by = ''"
	if rows(t, rt, exact) != 1 || rows(t, rt, "WHERE task_id = 't2'") != 0 {
		t.Fatal("the request must be pending, bound to the task the server named, and nothing may be decided")
	}

	if len(rt.grants.IDs()) != 0 {
		t.Fatal("nothing may be enforced before a person approves")
	}
}

func TestAskingAgainGivesTheSameAnswerAndNoNewRow(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	rt := startEgress(t, db, egressHub(t), dir)
	tools := &agentTools{egress: rt}
	input := `{"host":"marimo.example.com","reason":"x"}`

	askFor(tools, "t1", input)

	if again := askFor(tools, "t1", input); again.IsError || !strings.Contains(again.Output, "already asked") {
		t.Fatalf("%+v", again)
	}

	approved(t, rt, "t2", "other.example.com")

	res := askFor(tools, "t2", `{"host":"other.example.com","reason":"x"}`)
	if res.IsError || !strings.Contains(res.Output, "is allowed") {
		t.Fatalf("an approved host: %+v", res)
	}

	if rows(t, rt, "") != 2 {
		t.Fatalf("%d rows", rows(t, rt, ""))
	}
}

func TestBadInputIsRefusedAndSaysWhy(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	tools := &agentTools{egress: startEgress(t, db, egressHub(t), dir)}

	for name, input := range map[string]string{
		"ip": `{"host":"127.0.0.1","reason":"x"}`, "wildcard": `{"host":"*.example.com","reason":"x"}`,
		"port": `{"host":"a.example.com:443","reason":"x"}`, "url": `{"host":"https://a.example.com/x","reason":"x"}`,
		"internal": `{"host":"db.internal","reason":"x"}`, "no reason": `{"host":"a.example.com","reason":" "}`,
		"long reason": `{"host":"a.example.com","reason":"` + strings.Repeat("x", 600) + `"}`,
		"not json":    `host=a.example.com`, "no host": `{"reason":"x"}`,
	} {
		if res := askFor(tools, "t1", input); !res.IsError {
			t.Errorf("%s was accepted: %+v", name, res)
		}
	}

	if rows(t, tools.egress, "") != 0 {
		t.Fatal("nothing may be saved")
	}
}

func TestSpamIsStopped(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	rt := startEgress(t, db, egressHub(t), dir)
	tools := &agentTools{egress: rt}

	var refused int

	for i := range 12 {
		input := `{"host":"h` + string(rune('a'+i)) + `.example.com","reason":"x"}`
		if res := askFor(tools, "t1", input); res.IsError {
			refused++
		}
	}

	if rows(t, rt, "WHERE task_id = 't1'") != 5 || refused != 7 {
		t.Fatalf("%d saved, %d refused: five may wait at once", rows(t, rt, "WHERE task_id = 't1'"), refused)
	}

	if res := askFor(tools, "t2", `{"host":"ok.example.com","reason":"x"}`); res.IsError {
		t.Fatalf("another task is not affected: %+v", res)
	}
}

func TestConcurrentCallsForOneHostMakeOneRequest(t *testing.T) {
	db, dir := egressDB(t), t.TempDir()
	rt := startEgress(t, db, egressHub(t), dir)
	tools := &agentTools{egress: rt}

	var wg sync.WaitGroup

	for range 20 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			askFor(tools, "t1", `{"host":"same.example.com","reason":"x"}`)
		}()
	}

	wg.Wait()

	if rows(t, rt, "") != 1 {
		t.Fatalf("%d rows", rows(t, rt, ""))
	}
}

func TestTheToolHasNoWayToDecideAndFollowsRunCommand(t *testing.T) {
	if codingdomain.SwitchOf("request_egress") != "run_command" {
		t.Fatal("asking for a website must follow the run_command switch")
	}

	for _, info := range codingdomain.Tools() {
		if info.Name == "request_egress" {
			t.Fatal("outside AI apps must never be offered this tool")
		}
	}

	tools := &agentTools{}
	if tools.Risk("request_egress") != taskdomain.RiskNone {
		t.Fatal("it changes nothing until a person decides, so it needs no approval of its own")
	}
}
