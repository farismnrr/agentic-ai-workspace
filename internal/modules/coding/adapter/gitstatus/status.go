package gitstatus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	metadata, err := os.Lstat(filepath.Join(r.root, ".git"))
	if errors.Is(err, os.ErrNotExist) {
		return emptySnapshot(), nil
	}
	if err != nil || !metadata.IsDir() {
		return codingdomain.GitSnapshot{}, errors.New("unsupported git metadata")
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	output := &boundedBuffer{}
	cmd := exec.CommandContext(ctx, "git", "-C", r.root, "-c", "core.fsmonitor=false", "status",
		"--porcelain=v2", "-z", "--branch", "--untracked-files=normal", "--ignore-submodules=none")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0"}
	cmd.Stdout = output
	cmd.Stderr = &boundedBuffer{}
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return codingdomain.GitSnapshot{}, errors.New("git status timed out")
		}
		return codingdomain.GitSnapshot{}, fmt.Errorf("git status failed: %w", err)
	}

	return parse(output.Bytes())
}

func emptySnapshot() codingdomain.GitSnapshot {
	return codingdomain.GitSnapshot{Available: false, Files: []codingdomain.GitFile{}}
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > maxOutput {
		return 0, errors.New("git status output limit exceeded")
	}
	return b.Buffer.Write(p)
}

func parse(data []byte) (codingdomain.GitSnapshot, error) {
	parts := bytes.Split(data, []byte{0})
	out := codingdomain.GitSnapshot{Available: true, Files: make([]codingdomain.GitFile, 0)}
	for i := 0; i < len(parts); i++ {
		line := string(parts[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			parseHeader(line, &out)
			continue
		}
		file, rename, err := parseRecord(line)
		if err != nil {
			return codingdomain.GitSnapshot{}, err
		}
		if rename {
			i++
			if i >= len(parts) || len(parts[i]) == 0 {
				return codingdomain.GitSnapshot{}, errors.New("incomplete git rename record")
			}
			file.Original = string(parts[i])
		}
		if !codingdomain.Denied(file.Path) && !codingdomain.Denied(file.Original) {
			out.Files = append(out.Files, file)
		}
		if len(out.Files) > maxFiles {
			return codingdomain.GitSnapshot{}, errors.New("git status file limit exceeded")
		}
	}
	return out, nil
}

func parseHeader(line string, out *codingdomain.GitSnapshot) {
	key, value, found := strings.Cut(strings.TrimPrefix(line, "# "), " ")
	if !found {
		return
	}
	switch key {
	case "branch.oid":
		if value != "(initial)" {
			out.Revision = value
		} else {
			out.Initial = true
		}
	case "branch.head":
		if value != "(detached)" {
			out.Branch = value
		}
	}
}

func parseRecord(line string) (codingdomain.GitFile, bool, error) {
	fields := strings.SplitN(line, " ", 2)
	if len(fields) != 2 {
		return codingdomain.GitFile{}, false, errors.New("invalid git status record")
	}
	if fields[0] == "?" {
		return codingdomain.GitFile{Path: strings.TrimPrefix(fields[1], " "), Untracked: true}, false, nil
	}
	if fields[0] != "1" && fields[0] != "2" && fields[0] != "u" {
		return codingdomain.GitFile{}, false, errors.New("unknown git status record")
	}
	columnCount := 8
	if fields[0] == "2" {
		columnCount = 9
	} else if fields[0] == "u" {
		columnCount = 10
	}
	columns := strings.SplitN(fields[1], " ", columnCount)
	if len(columns) < columnCount || len(columns[0]) != 2 {
		return codingdomain.GitFile{}, false, errors.New("invalid git status columns")
	}
	path := columns[columnCount-1]
	status := codingdomain.GitFile{Path: path, Index: code(columns[0][0]), Worktree: code(columns[0][1]),
		Conflict: fields[0] == "u", Submodule: strings.HasPrefix(columns[1], "S")}
	return status, fields[0] == "2", nil
}

func code(value byte) string {
	if value == '.' || value == ' ' {
		return ""
	}
	return string(value)
}
