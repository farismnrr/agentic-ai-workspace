package gitstatus

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (c *metadataCopier) copyFile(path string, limit int64) ([]byte, error) {
	data, info, err := c.readSourceFile(path, limit)
	if err != nil {
		return nil, err
	}
	if addErr := c.addBytes(int64(len(data))); addErr != nil {
		return nil, addErr
	}
	if writeErr := writeSnapshotFile(c.target, path, data); writeErr != nil {
		return nil, writeErr
	}
	c.files[path] = info
	return data, nil
}

func (c *metadataCopier) readSourceFile(path string, limit int64) ([]byte, os.FileInfo, error) {
	file, info, err := c.openMetadataFile(path, limit)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := readAndVerifyMetadataFile(c.source, file, path, info, limit)
	if err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

func (c *metadataCopier) openMetadataFile(path string, limit int64) (*os.File, os.FileInfo, error) {
	info, err := c.source.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect Git metadata file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit || linkedFile(info) {
		return nil, nil, errGitMetadataEntry
	}
	if countErr := c.countEntry(); countErr != nil {
		return nil, nil, countErr
	}
	file, err := c.source.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open Git metadata file: %w", err)
	}
	opened, err := file.Stat()
	if err != nil || !sameMetadata(info, opened) {
		_ = file.Close()
		return nil, nil, errGitMetadataRace
	}
	return file, info, nil
}

func readAndVerifyMetadataFile(
	source *os.Root, file *os.File, path string, info os.FileInfo, limit int64,
) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read Git metadata file: %w", err)
	}
	if int64(len(data)) > limit || int64(len(data)) != info.Size() {
		return nil, errGitSnapshotLimit
	}
	after, statErr := file.Stat()
	current, pathErr := source.Lstat(path)
	if statErr != nil || pathErr != nil || !sameMetadata(info, after) || !sameMetadata(info, current) {
		return nil, errGitMetadataRace
	}
	return data, nil
}

func writeSnapshotFile(root, path string, data []byte) error {
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), privateDirectoryMode); err != nil {
		return fmt.Errorf("create Git snapshot directory: %w", err)
	}
	if err := os.WriteFile(target, data, privateFileMode); err != nil {
		return fmt.Errorf("write Git snapshot file: %w", err)
	}
	return nil
}
