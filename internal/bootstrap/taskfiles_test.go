package bootstrap

import (
	"encoding/json"
	"testing"
)

type changedFile struct {
	Path    string `json:"path"`
	OldPath string `json:"oldPath"`
	Status  string `json:"status"`
	Before  string `json:"before"`
	After   string `json:"after"`
}

func (a agentRig) changes(t *testing.T, id string) map[string]changedFile {
	t.Helper()

	var list []changedFile

	body := do(t, a.env.app, "GET", "/api/tasks/"+id+"/changes", "", jsonCookie(a.cookie)).Body
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

	a := newAgentRig(t,
		plan(),
		toolCall("e1", "edit_file", map[string]any{"path": "a.txt", "old_text": "two", "new_text": "three"}),
		toolCall("e2", "edit_file", map[string]any{"path": "a.txt", "old_text": "one", "new_text": "uno"}),
		toolCall("w1", "write_file", map[string]any{"path": "new.txt", "content": "fresh"}),
		toolCall("d1", "delete_file", map[string]any{"path": "gone.txt"}),
		toolCall("w2", "write_file", map[string]any{"path": "temp.txt", "content": "short lived"}),
		toolCall("d2", "delete_file", map[string]any{"path": "temp.txt"}),
		toolCall("m1", "move_file", map[string]any{"from": "keep.txt", "to": "moved.txt"}),
		finishCall("Done"),
	)
	a.put(t, "a.txt", "one\ntwo\n")
	a.put(t, "gone.txt", "bye")
	a.put(t, "keep.txt", "same")

	id := a.start(t, "Change some files")
	a.waitApproving(t, id, "review")

	changes := a.changes(t, id)

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
