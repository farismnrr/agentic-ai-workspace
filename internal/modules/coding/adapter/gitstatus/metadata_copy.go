package gitstatus

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"syscall"
)

const (
	maxGitMetadataFileBytes = 64 << 20
	maxGitMetadataBytes     = 256 << 20
	maxGitMetadataEntries   = 16384
)

type metadataCopier struct {
	source  *os.Root
	files   map[string]os.FileInfo
	dirs    map[string]metadataDirectory
	target  string
	bytes   int64
	entries int
}

type metadataDirectory struct {
	info  os.FileInfo
	names []string
}

func newMetadataCopier(source *os.Root, target string) *metadataCopier {
	return &metadataCopier{
		source: source, target: target, files: make(map[string]os.FileInfo),
		dirs: make(map[string]metadataDirectory),
	}
}

func (c *metadataCopier) copyDirectory(path string) ([]fs.DirEntry, error) {
	info, err := c.metadataDirectoryInfo(path)
	if err != nil {
		return nil, err
	}
	entries, err := c.readMetadataDirectory(path, info)
	if err != nil {
		return nil, err
	}
	if countErr := c.countEntry(); countErr != nil {
		return nil, countErr
	}
	if recordErr := c.recordDirectory(path, info, entries); recordErr != nil {
		return nil, recordErr
	}
	return entries, nil
}

func (c *metadataCopier) metadataDirectoryInfo(path string) (os.FileInfo, error) {
	info, err := c.source.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect Git metadata directory: %w", err)
	}
	if !isMetadataDirectory(info) {
		return nil, errGitMetadataEntry
	}
	return info, nil
}

func (c *metadataCopier) readMetadataDirectory(path string, info os.FileInfo) ([]fs.DirEntry, error) {
	directory, err := c.openMetadataDirectory(path, info)
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	entries, err := fs.ReadDir(directory.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("list Git metadata directory: %w", err)
	}
	return entries, nil
}

func (c *metadataCopier) recordDirectory(path string, info os.FileInfo, entries []fs.DirEntry) error {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !validMetadataName(entry.Name()) {
			return errGitMetadataEntry
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	c.dirs[path] = metadataDirectory{info: info, names: names}
	return nil
}

func (c *metadataCopier) openMetadataDirectory(path string, expected os.FileInfo) (*os.Root, error) {
	directory, err := c.source.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("open Git metadata directory: %w", err)
	}
	opened, err := directory.Stat(".")
	if err != nil || !sameMetadata(expected, opened) {
		_ = directory.Close()
		return nil, errGitMetadataRace
	}
	return directory, nil
}

func isMetadataDirectory(info os.FileInfo) bool {
	return info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func (c *metadataCopier) verify() error {
	for path, expected := range c.files {
		actual, err := c.source.Lstat(path)
		if err != nil || !sameMetadata(expected, actual) {
			return errGitMetadataRace
		}
	}
	for path, expected := range c.dirs {
		actual, err := c.source.Lstat(path)
		if err != nil || !sameMetadata(expected.info, actual) {
			return errGitMetadataRace
		}
		entries, err := fs.ReadDir(c.source.FS(), path)
		if err != nil || !sameNames(expected.names, entries) {
			return errGitMetadataRace
		}
	}
	return nil
}

func (c *metadataCopier) countEntry() error {
	c.entries++
	if c.entries > maxGitMetadataEntries {
		return errGitSnapshotLimit
	}
	return nil
}

func (c *metadataCopier) addBytes(size int64) error {
	if size < 0 || c.bytes+size > maxGitMetadataBytes {
		return errGitSnapshotLimit
	}
	c.bytes += size
	return nil
}

func sameMetadata(left, right os.FileInfo) bool {
	return os.SameFile(left, right) && left.Mode() == right.Mode() && left.Size() == right.Size() &&
		left.ModTime().Equal(right.ModTime())
}

func sameNames(expected []string, entries []fs.DirEntry) bool {
	if len(expected) != len(entries) {
		return false
	}
	for index, entry := range entries {
		if expected[index] != entry.Name() {
			return false
		}
	}
	return true
}

func validMetadataName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

func linkedFile(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}
