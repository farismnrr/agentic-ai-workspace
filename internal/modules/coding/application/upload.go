package application

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const uploadSniffBytes = 1024

type Stored struct {
	Path  string
	Name  string
	Kind  string
	Media string
	Size  int64
}

// Upload validates and stores one file without recording it as an undoable edit.
func (e *Editor) Upload(given string, size int64, src io.Reader) (Stored, error) {
	reader := bufio.NewReaderSize(src, uploadSniffBytes)
	head, err := reader.Peek(uploadSniffBytes)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return Stored{}, fmt.Errorf("read upload header: %w", err)
	}
	if len(head) == 0 {
		return Stored{}, domain.ErrEmpty
	}

	upload, check, err := domain.Classify(given, head)
	if err != nil {
		return Stored{}, fmt.Errorf("classify upload: %w", err)
	}
	if size > upload.Limit {
		return Stored{}, domain.ErrTooLarge
	}

	temp, err := e.createUpload(reader, upload.Limit, check)
	if err != nil {
		return Stored{}, err
	}
	defer func() { _ = e.writer.Remove(temp.path) }()

	if upload.Kind == "archive" {
		if err := e.validateArchive(temp.path, temp.size); err != nil {
			return Stored{}, fmt.Errorf("validate ZIP: %w", err)
		}
	}

	return e.finishUpload(given, upload, temp)
}

type stagedUpload struct {
	path string
	size int64
}

type byteCounter struct{ size int64 }

func (c *byteCounter) Write(data []byte) (int, error) {
	c.size += int64(len(data))
	return len(data), nil
}

func (e *Editor) createUpload(src io.Reader, limit int64, check domain.Check) (stagedUpload, error) {
	id, err := e.ids.ID()
	if err != nil {
		return stagedUpload{}, fmt.Errorf("create upload id: %w", err)
	}
	temp := "uploads/." + id + ".part"
	counter := &byteCounter{}
	copied := io.LimitReader(io.TeeReader(src, io.MultiWriter(check, counter)), limit+1)
	err = e.writer.Create(temp, copied)
	if err != nil {
		return stagedUpload{}, fmt.Errorf("write upload: %w", err)
	}
	if counter.size > limit {
		_ = e.writer.Remove(temp)
		return stagedUpload{}, domain.ErrTooLarge
	}
	if err := check.Err(); err != nil {
		_ = e.writer.Remove(temp)
		return stagedUpload{}, fmt.Errorf("validate upload content: %w", err)
	}
	return stagedUpload{path: temp, size: counter.size}, nil
}

func (e *Editor) finishUpload(given string, upload domain.Upload, staged stagedUpload) (Stored, error) {
	if staged.size > upload.Limit {
		return Stored{}, domain.ErrTooLarge
	}
	base := domain.SafeBase(given, "file")
	e.mu.Lock()
	defer e.mu.Unlock()
	for suffix := 1; suffix <= 100; suffix++ {
		name := uploadName(base, upload.Ext, suffix)
		target := path.Join("uploads", name)
		if _, err := e.files.Stat(target); err == nil {
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			return Stored{}, fmt.Errorf("check upload name: %w", err)
		}
		file, _, err := e.files.Open(staged.path)
		if err != nil {
			return Stored{}, fmt.Errorf("open staged upload: %w", err)
		}
		err = e.writer.Create(target, file)
		_ = file.Close()
		if errors.Is(err, domain.ErrExists) {
			continue
		}
		if err != nil {
			return Stored{}, fmt.Errorf("finish upload: %w", err)
		}
		return Stored{Path: target, Name: name, Kind: upload.Kind, Media: upload.Media, Size: staged.size}, nil
	}
	return Stored{}, domain.ErrExists
}

func uploadName(base, ext string, suffix int) string {
	if suffix == 1 {
		return base + ext
	}
	return fmt.Sprintf("%s-%d%s", base, suffix, ext)
}
