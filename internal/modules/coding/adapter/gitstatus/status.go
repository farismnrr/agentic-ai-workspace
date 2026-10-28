package gitstatus

import (
	"errors"
	"fmt"
	"os"
	"time"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	commandTimeout = 3 * time.Second
	maxOutput      = 2 << 20
	maxFiles       = 2000
)

// Reader runs read-only Git status within one authorized project directory.
type Reader struct{ root string }

// New binds the reader to a project directory resolved by the project folder adapter.
func New(root string) *Reader { return &Reader{root: root} }

// Read returns an unavailable snapshot for non-repositories and bounded status otherwise.
func (r *Reader) Read() (codingdomain.GitSnapshot, error) {
	snapshot, err := createMetadataSnapshot(r.root)
	if errors.Is(err, os.ErrNotExist) {
		return emptySnapshot(), nil
	}
	if err != nil {
		return codingdomain.GitSnapshot{}, err
	}
	result, readErr := readStatusSnapshot(snapshot)
	closeErr := snapshot.Close()
	if readErr != nil {
		return codingdomain.GitSnapshot{}, readErr
	}
	if closeErr != nil {
		return codingdomain.GitSnapshot{}, closeErr
	}
	return result, nil
}

func readStatusSnapshot(snapshot *metadataSnapshot) (codingdomain.GitSnapshot, error) {
	output, err := runGit(snapshot, maxOutput+1, "status", "--porcelain=v2", "-z", "--branch",
		"--untracked-files=normal", "--ignore-submodules=all")
	if err != nil {
		if errors.Is(err, errGitCommandTimeout) {
			return codingdomain.GitSnapshot{}, errGitStatusTimeout
		}
		return codingdomain.GitSnapshot{}, fmt.Errorf("git status failed: %w", err)
	}
	if len(output) > maxOutput {
		return codingdomain.GitSnapshot{}, errGitStatusOutputLimit
	}
	return parse(output)
}

func emptySnapshot() codingdomain.GitSnapshot {
	return codingdomain.GitSnapshot{Available: false, Files: []codingdomain.GitFile{}}
}
