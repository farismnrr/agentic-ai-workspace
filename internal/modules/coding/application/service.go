package application

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const binarySniffBytes = 8000

// FileSystem is the read-only view of the workspace the use cases need. An
// implementation must refuse to reach outside the workspace root.
type FileSystem interface {
	// Stat describes one path; a missing path returns domain.ErrNotFound.
	Stat(rel string) (domain.Entry, error)
	ReadDir(rel string) ([]domain.Entry, error)
	// ReadFile returns at most max bytes and whether more were available.
	ReadFile(rel string, max int64) (data []byte, truncated bool, err error)
	// Open returns a confined, seekable file and its size; directories are refused.
	Open(rel string) (io.ReadSeekCloser, int64, error)
	// Walk visits files under rel; returning ErrSkipDir from visit skips a directory.
	Walk(rel string, visit func(entry domain.Entry) error) error
}

// Service holds the read-only coding use cases.
type Service struct {
	files FileSystem
}

// NewService wires the use cases to a workspace.
func NewService(files FileSystem) *Service {
	return &Service{files: files}
}

// Listing is a directory listing, possibly cut short.
type Listing struct {
	Entries   []domain.Entry
	Truncated bool
}

// List returns the visible entries of a directory, directories first.
func (s *Service) List(raw string) (Listing, error) {
	rel, err := domain.CleanPath(raw)
	if err != nil {
		return Listing{}, fmt.Errorf("list: %w", err)
	}

	entries, err := s.files.ReadDir(rel)
	if err != nil {
		return Listing{}, fmt.Errorf("list %s: %w", rel, err)
	}

	visible := entries[:0]

	for _, entry := range entries {
		if !domain.Denied(entry.Path) {
			visible = append(visible, entry)
		}
	}

	sort.Slice(visible, func(i, j int) bool {
		if visible[i].Dir != visible[j].Dir {
			return visible[i].Dir
		}

		return visible[i].Path < visible[j].Path
	})

	if len(visible) > domain.MaxListEntries {
		return Listing{Entries: visible[:domain.MaxListEntries], Truncated: true}, nil
	}

	return Listing{Entries: visible}, nil
}

// File is the text of a file (or a line range of it), possibly cut short.
type File struct {
	Path       string
	Content    string
	TotalLines int
	Truncated  bool
}

// Read returns a whole text file, refusing binaries.
func (s *Service) Read(raw string) (File, error) {
	return s.ReadLines(raw, 0, 0)
}

// ReadLines returns lines start..end (1-based, inclusive) of a text file; zero
// values mean "from the first line" and "to the last".
func (s *Service) ReadLines(raw string, start, end int) (File, error) {
	if start < 0 || end < 0 || (end > 0 && start > end) {
		return File{}, domain.ErrBadRange
	}

	rel, err := domain.CleanPath(raw)
	if err != nil {
		return File{}, fmt.Errorf("read: %w", err)
	}

	data, truncated, err := s.files.ReadFile(rel, domain.MaxReadBytes)
	if err != nil {
		return File{}, fmt.Errorf("read %s: %w", rel, err)
	}

	if isBinary(data) {
		return File{}, domain.ErrBinary
	}

	text := strings.ToValidUTF8(string(data), "\uFFFD")
	file := File{Path: rel, Content: text, Truncated: truncated, TotalLines: strings.Count(text, "\n") + 1}

	if start > 0 || end > 0 {
		file.Content = sliceLines(text, start, end)
	}

	return file, nil
}

func sliceLines(text string, start, end int) string {
	lines := strings.Split(text, "\n")
	start = max(start, 1)

	if end == 0 || end > len(lines) {
		end = len(lines)
	}

	if start > end {
		return ""
	}

	return strings.Join(lines[start-1:end], "\n")
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data[:min(len(data), binarySniffBytes)], 0) >= 0
}
