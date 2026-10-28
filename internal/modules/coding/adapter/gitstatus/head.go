package gitstatus

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	maxHeadBytes        = codingdomain.MaxReadBytes
	maxHeadObjectSize   = 4 * maxHeadBytes
	treeRecordLimit     = 4096
	sizeOutputLimit     = 64
	treeEntryParts      = 2
	revisionOutputLimit = 1024
)

// ReadHead returns only a small text blob from this repository's HEAD.
func (r *Reader) ReadHead(rawPath string) (codingdomain.GitHead, error) {
	path, err := codingdomain.GitPath(rawPath)
	if err != nil {
		return codingdomain.GitHead{}, fmt.Errorf("validate Git HEAD path: %w", err)
	}
	snapshot, err := createMetadataSnapshot(r.root)
	if errors.Is(err, os.ErrNotExist) {
		return codingdomain.GitHead{Available: false}, nil
	}
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	head, readErr := readHeadSnapshot(snapshot, path)
	closeErr := snapshot.Close()
	if readErr != nil {
		return codingdomain.GitHead{}, readErr
	}
	if closeErr != nil {
		return codingdomain.GitHead{}, closeErr
	}
	return head, nil
}

func readHeadSnapshot(snapshot *metadataSnapshot, path string) (codingdomain.GitHead, error) {
	available, initial, err := headState(snapshot)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	if !available || initial {
		return codingdomain.GitHead{Available: available, Initial: initial}, nil
	}
	object, err := headObject(snapshot, path)
	if err != nil || object == "" {
		return codingdomain.GitHead{Available: true, Content: ""}, err
	}
	return readHeadBlob(snapshot, object)
}

func headState(snapshot *metadataSnapshot) (bool, bool, error) {
	if _, err := runGit(snapshot, revisionOutputLimit, "rev-parse", "--verify", "HEAD^{commit}"); err == nil {
		return true, false, nil
	} else if errors.Is(err, errGitCommandTimeout) {
		return false, false, errGitHeadUnavailable
	}
	status, err := readStatusSnapshot(snapshot)
	if err != nil {
		return false, false, fmt.Errorf("check repository without HEAD: %w", err)
	}
	if !status.Available {
		return false, false, nil
	}
	if status.Initial {
		return true, true, nil
	}
	return false, false, errGitHeadUnavailable
}

func headObject(snapshot *metadataSnapshot, path string) (string, error) {
	tree, err := runGit(snapshot, treeRecordLimit, "ls-tree", "-z", "HEAD", "--", ":(literal)"+path)
	if err != nil {
		return "", fmt.Errorf("read Git HEAD tree: %w", err)
	}
	return parseHeadObject(tree, path, snapshot.objectFormat)
}

func readHeadBlob(snapshot *metadataSnapshot, object string) (codingdomain.GitHead, error) {
	size, err := headBlobSize(snapshot, object)
	if err != nil {
		return codingdomain.GitHead{}, err
	}
	content, err := runGit(snapshot, maxHeadBytes+1, "cat-file", "blob", object)
	if err != nil {
		return codingdomain.GitHead{}, fmt.Errorf("read Git HEAD blob: %w", err)
	}
	return textHead(content, size), nil
}

func headBlobSize(snapshot *metadataSnapshot, object string) (int64, error) {
	output, err := runGit(snapshot, sizeOutputLimit, "cat-file", "-s", object)
	if err != nil {
		return 0, fmt.Errorf("read Git HEAD blob size: %w", err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil || size < 0 {
		return 0, errGitBlobSize
	}
	if size > maxHeadObjectSize {
		return 0, errGitBlobLimit
	}
	return size, nil
}

func textHead(content []byte, size int64) codingdomain.GitHead {
	truncated := size > maxHeadBytes
	if len(content) > maxHeadBytes {
		content = utf8Boundary(content[:maxHeadBytes])
	}
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return codingdomain.GitHead{Available: true, Binary: true}
	}
	return codingdomain.GitHead{Available: true, Content: string(content), Truncated: truncated}
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

func parseHeadObject(data []byte, wanted, format string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	entry := strings.SplitN(string(bytes.TrimSuffix(data, []byte{0})), "\t", treeEntryParts)
	if len(entry) != treeEntryParts {
		return "", errGitTreeRecord
	}
	fields := strings.Fields(entry[0])
	if entry[1] != wanted || len(fields) != 3 || fields[1] != "blob" ||
		len(fields[2]) != objectIDLength(format) || !lowerHex(fields[2]) {
		return "", codingdomain.ErrIsDir
	}
	return fields[2], nil
}
