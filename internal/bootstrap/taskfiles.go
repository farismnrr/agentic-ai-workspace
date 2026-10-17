package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"unicode/utf8"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// binarySniffBytes is how much of a backup is looked at to tell text from a binary file.
const binarySniffBytes = 8000

// fileOrigin is what a file held before the task first touched it.
type fileOrigin struct {
	before  string
	from    string
	existed bool
	binary  bool
	// same means nothing is known to differ from the file now, as for a plain rename.
	same bool
}

// TaskFiles rebuilds what a task changed, file by file: the oldest backup of each file is what it held before
// the task, the file on disk is what it holds now. Edits of one file merge into one entry, the way a pull request
// shows them; a file created and removed again, or put back as it was, is left out.
func (a *agentTools) TaskFiles(_ context.Context, projectID string, blobs [][]byte) ([]taskapp.FileDiff, error) {
	services, err := a.services(projectID)
	if err != nil {
		return nil, err
	}

	origins, order, moved := a.origins(blobs)
	diffs := make([]taskapp.FileDiff, 0, len(order))

	for _, path := range order {
		if moved[path] {
			continue
		}

		if diff, keep := compare(path, origins[path], services); keep {
			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
}

// origins reads the recorded changes in order and remembers the first state of every file they touched. A file
// moved away is listed under its new name only.
func (a *agentTools) origins(blobs [][]byte) (map[string]fileOrigin, []string, map[string]bool) {
	origins := map[string]fileOrigin{}
	moved := map[string]bool{}

	var order []string

	for _, blob := range blobs {
		change, err := codingdomain.ParseChange(blob)
		if err != nil {
			continue
		}

		if change.Op == codingdomain.OpMove {
			from, known := origins[change.Path]
			if !known {
				from = fileOrigin{existed: true, same: true}
			}

			from.from = change.Path
			moved[change.Path] = true
			origins[change.To] = from
			order = append(order, change.To)

			continue
		}

		if _, seen := origins[change.Path]; !seen {
			origins[change.Path] = a.originOf(change)
			order = append(order, change.Path)
		}
	}

	return origins, order, moved
}

func (a *agentTools) originOf(change codingdomain.Change) fileOrigin {
	origin := fileOrigin{existed: change.Existed || change.Op == codingdomain.OpDelete}
	if change.BackupID == "" {
		return origin
	}

	data, err := a.backups.Load(change.BackupID)
	if err != nil {
		return origin
	}

	origin.binary = !utf8.Valid(data) || bytes.ContainsRune(data[:min(len(data), binarySniffBytes)], 0)
	origin.before = string(data)

	return origin
}

// compare sets one file's origin against the file as it is now, and says whether it differs at all.
func compare(path string, origin fileOrigin, services codingmcp.Services) (taskapp.FileDiff, bool) {
	file, err := services.Reader.Read(path)
	exists := err == nil || errors.Is(err, codingdomain.ErrBinary)
	binary := origin.binary || errors.Is(err, codingdomain.ErrBinary)
	diff := taskapp.FileDiff{Path: path, OldPath: origin.from, Before: origin.before, After: file.Content, Binary: binary}

	if origin.same {
		diff.Before = file.Content
	}

	if len(diff.Before) > codingdomain.MaxReadBytes {
		diff.Before, diff.Truncated = diff.Before[:codingdomain.MaxReadBytes], true
	}

	diff.Truncated = diff.Truncated || file.Truncated

	diff.Status = statusOf(origin, exists, diff)
	if diff.Status == "" {
		return diff, false
	}

	if diff.Status == "deleted" {
		diff.After = ""
	}

	if binary {
		diff.Before, diff.After = "", ""
	}

	return diff, true
}

// statusOf says how a file differs from how the task found it: added, modified, deleted or renamed, or nothing
// when it is the same (or never existed at either end).
func statusOf(origin fileOrigin, exists bool, diff taskapp.FileDiff) string {
	switch {
	case !origin.existed && !exists:
		return ""
	case !origin.existed:
		return "added"
	case !exists:
		return "deleted"
	case origin.from != "":
		return "renamed"
	case diff.Before == diff.After && !diff.Binary:
		return ""
	}

	return "modified"
}
