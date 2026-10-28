package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

const effectHashChunkSize = 32 << 10

var errSnapshotFileChanged = errors.New("snapshot file changed")

func (s *effectScanner) digest(
	ctx context.Context,
	name string,
	expected os.FileInfo,
) (digest string, resultErr error) {
	if expected.Size() < 0 || expected.Size() > maxEffectBytes-s.state.counters.bytes {
		return "", errEffectSnapshotLimit
	}
	file, err := s.state.root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", fmt.Errorf("open snapshot file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close snapshot file: %w", closeErr))
			digest = ""
		}
	}()
	if checkErr := verifyOpenFile(file, expected); checkErr != nil {
		return "", fmt.Errorf("%w before hashing: %s", errSnapshotFileChanged, name)
	}
	hash := sha256.New()
	buffer := make([]byte, effectHashChunkSize)
	count, err := s.hashFile(ctx, file, name, buffer, hash)
	if err != nil {
		return "", err
	}
	if count != expected.Size() || verifyOpenFile(file, expected) != nil {
		return "", fmt.Errorf("%w while hashing: %s", errSnapshotFileChanged, name)
	}
	if !s.pathMatches(name, expected) {
		return "", fmt.Errorf("%w: %s", errSnapshotFileChanged, name)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *effectScanner) pathMatches(name string, expected os.FileInfo) bool {
	current, err := s.state.root.Lstat(name)
	return err == nil && sameMetadata(expected, current)
}

func verifyOpenFile(file *os.File, expected os.FileInfo) error {
	opened, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat open snapshot file: %w", err)
	}
	if !sameMetadata(expected, opened) {
		return errSnapshotFileChanged
	}
	return nil
}

func (s *effectScanner) hashFile(
	ctx context.Context,
	file *os.File,
	name string,
	buffer []byte,
	hash io.Writer,
) (int64, error) {
	var count int64
	var tail []byte
	for {
		read, readErr, err := s.readSnapshotChunk(ctx, file, buffer)
		if err != nil {
			return 0, err
		}
		count += int64(read)
		if read > 0 {
			_, _ = hash.Write(buffer[:read])
			if strings.HasPrefix(name, ".git/config") {
				tail = scanGitConfig(s, tail, buffer[:read])
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
	}
	return count, nil
}

func (s *effectScanner) readSnapshotChunk(ctx context.Context, file *os.File, buffer []byte) (int, error, error) {
	if contextErr := ctx.Err(); contextErr != nil {
		return 0, nil, fmt.Errorf("hash snapshot file: %w", contextErr)
	}
	remaining := maxEffectBytes - s.state.counters.bytes
	if remaining == 0 {
		return 0, io.EOF, nil
	}
	chunk := buffer
	if int64(len(chunk)) > remaining {
		chunk = chunk[:remaining]
	}
	read, readErr := file.Read(chunk)
	s.state.counters.bytes += int64(read)
	if s.state.counters.bytes > maxEffectBytes {
		return 0, nil, errEffectSnapshotLimit
	}
	if readErr != nil {
		return read, fmt.Errorf("read snapshot file: %w", readErr), nil
	}
	return read, nil, nil
}

func scanGitConfig(scanner *effectScanner, tail, chunk []byte) []byte {
	combined := append(append([]byte(nil), tail...), chunk...)
	lower := strings.ToLower(string(combined))
	if strings.Contains(lower, "hookspath") || strings.Contains(lower, "[include") {
		scanner.state.counters.externalGitConfig = true
	}
	keep := len("hookspath") - 1
	if len(combined) > keep {
		return append(tail[:0], combined[len(combined)-keep:]...)
	}
	return append(tail[:0], combined...)
}
