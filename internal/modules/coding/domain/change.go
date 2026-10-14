package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Op is the kind of modification a tool made.
type Op string

// Modification kinds.
const (
	OpEdit   Op = "edit"
	OpWrite  Op = "write"
	OpDelete Op = "delete"
	OpMove   Op = "move"
)

// ErrChanged means the file no longer matches what the tool left behind, so
// undoing would overwrite someone else's later work.
var ErrChanged = errors.New("the file changed after this edit, so it was not undone")

// ErrNotUndoable means the record does not describe a reversible change.
var ErrNotUndoable = errors.New("this change cannot be undone")

// Change records how to reverse one modification. It is stored with the
// activity entry as an opaque JSON blob.
type Change struct {
	Op        Op     `json:"op"`
	Path      string `json:"path"`
	To        string `json:"to,omitempty"`
	BackupID  string `json:"backup,omitempty"`
	AfterHash string `json:"after,omitempty"`
	Existed   bool   `json:"existed,omitempty"`
}

// Encode serialises the change for storage.
func (c Change) Encode() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode change: %w", err)
	}

	return data, nil
}

// ParseChange reads a stored change.
func ParseChange(data []byte) (Change, error) {
	var change Change
	if err := json.Unmarshal(data, &change); err != nil || change.Op == "" || change.Path == "" {
		return Change{}, ErrNotUndoable
	}

	return change, nil
}
