package domain

import (
	"errors"
	"path"
	"strings"
)

// Limits keep every tool response small and every request cheap.
const (
	MaxListEntries     = 500
	MaxReadBytes       = 256 << 10
	MaxSearchMatches   = 100
	MaxSearchFileBytes = 1 << 20
	MaxQueryLength     = 200
	MaxMatchLineLength = 300
	MaxWriteBytes      = 1 << 20
	MaxFindResults     = 200
	MaxOutputBytes     = 64 << 10
)

// Errors a tool call can fail with; their text is safe to show to the model.
var (
	ErrDenied     = errors.New("that path is not available")
	ErrNotFound   = errors.New("no such file or directory")
	ErrIsDir      = errors.New("that path is a directory; use list_files")
	ErrNotDir     = errors.New("that path is a file; use read_file")
	ErrBinary     = errors.New("that file is binary and cannot be shown as text")
	ErrEmptyQuery = errors.New("query must be 1 to 200 characters")
	ErrTooLarge   = errors.New("content is larger than the 1 MB limit")
	ErrExists     = errors.New("something already exists at that path")
	ErrNotUnique  = errors.New("old_text appears more than once; add more context or set replace_all")
	ErrNoMatch    = errors.New("old_text was not found in the file")
	ErrBadRange   = errors.New("line range must be 1-based and start at or before its end")
	ErrBadPattern = errors.New("pattern is not a valid glob")
)

// Entry is one file or directory in a listing.
type Entry struct {
	Path string
	Dir  bool
	Size int64
}

// Match is one line that contains a search query.
type Match struct {
	Path string
	Text string
	Line int
}

// CleanPath turns user input into a workspace-relative slash path ("." is the
// root). Cleaning a rooted path first means ".." can never climb above it.
func CleanPath(raw string) (string, error) {
	if strings.ContainsRune(raw, 0) {
		return "", ErrDenied
	}

	rel := strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(raw, `\`, "/")), "/")
	if rel == "" {
		return ".", nil
	}

	if Denied(rel) {
		return "", ErrDenied
	}

	return rel, nil
}

// Denied reports whether a path holds secrets, VCS internals or the task definitions the tools must never reach.
func Denied(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		lower := strings.ToLower(segment)

		switch {
		case lower == ".git", lower == ".workspace", lower == ".env", strings.HasPrefix(lower, ".env."):
			return true
		case strings.HasSuffix(lower, ".pem"), strings.HasSuffix(lower, ".key"), strings.HasPrefix(lower, "id_rsa"):
			return true
		}
	}

	return false
}
