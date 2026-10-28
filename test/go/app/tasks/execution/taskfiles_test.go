package execution_test

import (
	"encoding/json"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

type changedFile struct {
	Path    string `json:"path"`
	OldPath string `json:"oldPath"`
	Status  string `json:"status"`
	Before  string `json:"before"`
	After   string `json:"after"`
}

func changes(t *testing.T, a apptest.AgentRig, id string) map[string]changedFile {
	t.Helper()

	var list []changedFile

	body := apptest.Do(t, a.Environment.App, "GET", "/api/tasks/"+id+"/changes", "", apptest.JSONCookie(a.Cookie)).Body
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("%v: %s", err, body)
	}

	out := map[string]changedFile{}
	for _, file := range list {
		out[file.Path] = file
	}

	return out
}

func TestATaskShowsEachChangedFileAsOneDiffAgainstTheStartOfTheTask(t *testing.T) {
	t.Parallel()

	a := apptest.NewAgentRig(t,
		apptest.Plan(),
		apptest.ToolCall("e1", "edit_file", map[string]any{"path": "a.txt", "old_text": "two", "new_text": "three"}),
		apptest.ToolCall("e2", "edit_file", map[string]any{"path": "a.txt", "old_text": "one", "new_text": "uno"}),
		apptest.ToolCall("w1", "write_file", map[string]any{"path": "new.txt", "content": "fresh"}),
		apptest.ToolCall("d1", "delete_file", map[string]any{"path": "gone.txt"}),
		apptest.ToolCall("w2", "write_file", map[string]any{"path": "temp.txt", "content": "short lived"}),
		apptest.ToolCall("d2", "delete_file", map[string]any{"path": "temp.txt"}),
		apptest.ToolCall("m1", "move_file", map[string]any{"from": "keep.txt", "to": "moved.txt"}),
		apptest.FinishCall("Done"),
	)
	a.PutFile(t, "a.txt", "one\ntwo\n")
	a.PutFile(t, "gone.txt", "bye")
	a.PutFile(t, "keep.txt", "same")

	id := a.Start(t, "Change some files")
	a.WaitApproving(t, id, "review")

	changes := changes(t, a, id)

	want := map[string]changedFile{
		"a.txt":     {Status: "modified", Before: "one\ntwo\n", After: "uno\nthree\n"},
		"new.txt":   {Status: "added", After: "fresh"},
		"gone.txt":  {Status: "deleted", Before: "bye"},
		"moved.txt": {Status: "renamed", OldPath: "keep.txt", Before: "same", After: "same"},
	}

	for path, expected := range want {
		got := changes[path]
		same := got.Status == expected.Status && got.OldPath == expected.OldPath
		if !same || got.Before != expected.Before || got.After != expected.After {
			t.Errorf("%s: got %+v, want %+v", path, got, expected)
		}
	}

	// Two edits are one change; a file made and removed again, and the old name of a moved file, are not changes.
	if len(changes) != len(want) {
		t.Fatalf("only the four real changes are listed: %v", changes)
	}
}
