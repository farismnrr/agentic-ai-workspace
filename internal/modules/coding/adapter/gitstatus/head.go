package gitstatus

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const maxHeadBytes = codingdomain.MaxReadBytes

// ReadHead returns only a small text blob from this repository's HEAD.
func (r *Reader) ReadHead(rawPath string) (codingdomain.GitHead, error) {
	path, err := codingdomain.GitPath(rawPath)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	metadata, err := os.Lstat(filepath.Join(r.root, ".git"))
	if errors.Is(err, os.ErrNotExist) {
		return codingdomain.GitHead{Available: false}, nil
	}
	if err != nil || !metadata.IsDir() {
		return codingdomain.GitHead{}, errors.New("unsupported git metadata")
	}

	if _, err := runGit(r.root, 1024, "rev-parse", "--verify", "HEAD^{commit}"); err != nil {
		snapshot, statusErr := r.Read()
		if statusErr != nil || !snapshot.Available {
			return codingdomain.GitHead{Available: false, Content: ""}, nil
		}
		if snapshot.Initial {
			return codingdomain.GitHead{Available: true, Initial: true, Content: ""}, nil
		}
		return codingdomain.GitHead{}, errors.New("git HEAD could not be read")
	}
	tree, err := runGit(r.root, 4096, "ls-tree", "-z", "HEAD", "--", ":(literal)"+path)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	object, err := headObject(tree, path)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	if object == "" {
		return codingdomain.GitHead{Available: true, Content: ""}, nil
	}
	content, err := runGit(r.root, maxHeadBytes+1, "cat-file", "blob", object)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return codingdomain.GitHead{Available: true, Binary: true}, nil
	}
	truncated := len(content) > maxHeadBytes
	if truncated {
		content = content[:maxHeadBytes]
	}
	return codingdomain.GitHead{Available: true, Content: string(content), Truncated: truncated}, nil
}

func headObject(data []byte, wanted string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	entry := strings.SplitN(string(bytes.TrimSuffix(data, []byte{0})), "\t", 2)
	if len(entry) != 2 {
		return "", errors.New("invalid git tree record")
	}
	if entry[1] != wanted {
		return "", codingdomain.ErrIsDir
	}
	fields := strings.Fields(entry[0])
	if len(fields) != 3 || fields[1] != "blob" {
		return "", codingdomain.ErrIsDir
	}
	return fields[2], nil
}

func runGit(root string, limit int, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	output := &prefixBuffer{limit: limit}
	commandArgs := append([]string{"-C", root, "-c", "core.fsmonitor=false"}, args...)
	cmd := exec.CommandContext(ctx, "git", commandArgs...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "GIT_ATTR_NOSYSTEM=1"}
	cmd.Stdout = output
	cmd.Stderr = &boundedBuffer{}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type prefixBuffer struct {
	bytes.Buffer
	limit int
}

func (b *prefixBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.Buffer.Write(p[:remaining])
		} else {
			_, _ = b.Buffer.Write(p)
		}
	}
	return len(p), nil
}
