package application

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	maxArchiveFiles    = 256
	maxArchiveUnpacked = 100 << 20
	maxArchiveRatio    = 200
	maxArchivePath     = 256
	maxArchiveDepth    = 12
)

var errUnsafeArchive = fmt.Errorf("%w: unsafe ZIP archive", domain.ErrUnsupported)

// validateArchive checks the contents without extracting any file onto the host.
func (e *Editor) validateArchive(relative string, size int64) error {
	data, err := e.readArchivedBytes(relative, size)
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(bytes.NewReader(data), size)
	if err != nil || len(reader.File) == 0 || len(reader.File) > maxArchiveFiles {
		return errUnsafeArchive
	}
	names := make(map[string]bool, len(reader.File))
	var total uint64
	for _, entry := range reader.File {
		if err := validateArchiveEntry(entry, names, &total); err != nil {
			return err
		}
	}
	return nil
}

func (e *Editor) readArchivedBytes(relative string, size int64) ([]byte, error) {
	if size < 22 || size > domain.MaxUploadBytes {
		return nil, errUnsafeArchive
	}
	file, _, err := e.files.Open(relative)
	if err != nil {
		return nil, fmt.Errorf("read ZIP: %w", err)
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, domain.MaxUploadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read ZIP: %w", err)
	}
	if int64(len(data)) != size {
		return nil, errUnsafeArchive
	}
	return data, nil
}

func validArchiveName(name string, seen map[string]bool) bool {
	clean := path.Clean(name)
	if invalidArchivePath(name, clean) || seen[clean] {
		return false
	}
	seen[clean] = true
	return true
}

func invalidArchivePath(name, clean string) bool {
	return len(name) > maxArchivePath || strings.ContainsAny(name, "\\\x00:") ||
		strings.HasPrefix(name, "/") || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, "../") || strings.Contains(name, "//") ||
		strings.Count(name, "/") > maxArchiveDepth || unsafeArchiveSegments(name)
}

func unsafeArchiveSegments(name string) bool {
	for _, part := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		if part == "." || part == ".." || part == "" {
			return true
		}
	}
	return false
}

func validateArchiveEntry(entry *zip.File, names map[string]bool, total *uint64) error {
	if !validArchiveName(entry.Name, names) || !supportedArchiveEntry(entry) ||
		!withinArchiveLimits(entry, *total) {
		return errUnsafeArchive
	}
	info := entry.FileInfo()
	unpacked := entry.UncompressedSize64
	*total += unpacked
	if info.IsDir() {
		return nil
	}
	return scanArchiveEntry(entry, unpacked)
}

func supportedArchiveEntry(entry *zip.File) bool {
	if entry.Flags&1 != 0 || (entry.Method != zip.Store && entry.Method != zip.Deflate) {
		return false
	}
	info := entry.FileInfo()
	return (info.IsDir() || info.Mode().IsRegular()) && (info.IsDir() || !nestedArchive(entry.Name))
}

func withinArchiveLimits(entry *zip.File, total uint64) bool {
	unpacked := entry.UncompressedSize64
	if unpacked > maxArchiveUnpacked || total+unpacked > maxArchiveUnpacked {
		return false
	}
	if unpacked > 0 && (entry.CompressedSize64 == 0 ||
		unpacked/entry.CompressedSize64 > maxArchiveRatio) {
		return false
	}
	return true
}

func nestedArchive(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".zip", ".jar", ".7z", ".rar", ".tar", ".gz", ".xz":
		return true
	}
	return false
}

func scanArchiveEntry(entry *zip.File, unpacked uint64) error {
	reader, err := entry.Open()
	if err != nil {
		return errUnsafeArchive
	}
	// unpacked is validated above to be <=100 MiB, far below MaxInt64.
	expected := int64(unpacked) //nolint:gosec // checked in validateArchiveEntry
	n, readErr := io.Copy(io.Discard, io.LimitReader(reader, expected+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || n != expected {
		return errUnsafeArchive
	}
	return nil
}
