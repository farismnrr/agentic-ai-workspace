package gitstatus

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	maxHeadBytes      = codingdomain.MaxReadBytes
	maxHeadObjectSize = 4 * maxHeadBytes
)

// ReadHead returns only a small text blob from this repository's HEAD.
func (r *Reader) ReadHead(rawPath string) (codingdomain.GitHead, error) {
	path, err := codingdomain.GitPath(rawPath)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	err = validateGitMetadata(r.root)
	if errors.Is(err, os.ErrNotExist) {
		return codingdomain.GitHead{Available: false}, nil
	}
	if err != nil {
		return codingdomain.GitHead{}, err
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
	sizeOutput, err := runGit(r.root, 64, "cat-file", "-s", object)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(sizeOutput)), 10, 64)
	if err != nil || size < 0 {
		return codingdomain.GitHead{}, errors.New("invalid git blob size")
	}
	if size > maxHeadObjectSize {
		return codingdomain.GitHead{}, errors.New("git blob exceeds read limit")
	}
	content, err := runGit(r.root, maxHeadBytes+1, "cat-file", "blob", object)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	truncated := size > maxHeadBytes
	if len(content) > maxHeadBytes {
		content = utf8Boundary(content[:maxHeadBytes])
	}
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return codingdomain.GitHead{Available: true, Binary: true}, nil
	}
	return codingdomain.GitHead{Available: true, Content: string(content), Truncated: truncated}, nil
}

func utf8Boundary(data []byte) []byte {
	start := len(data) - 1
	for start > 0 && data[start]&0xc0 == 0x80 {
		start--
	}
	if !utf8.FullRune(data[start:]) {
		return data[:start]
	}
	return data
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
	if err := validateGitMetadata(root); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	output := &prefixBuffer{limit: limit}
	cmd := exec.CommandContext(ctx, "git", gitCommandArgs(root, args...)...)
	cmd.Env = gitEnvironment()
	cmd.Stdout = output
	cmd.Stderr = &boundedBuffer{}
	cmd.WaitDelay = time.Second
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
