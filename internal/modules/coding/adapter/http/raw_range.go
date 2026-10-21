package http

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
)

type sectionStream struct {
	*io.SectionReader
	io.Closer
}

type seekReaderAt struct {
	file io.ReadSeekCloser
	mu   sync.Mutex
}

func (r *seekReaderAt) ReadAt(data []byte, offset int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek raw file: %w", err)
	}
	n, err := io.ReadFull(r.file, data)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = io.EOF
	}
	return n, err
}

func newSection(file application.RawStream, start, size int64) *io.SectionReader {
	return io.NewSectionReader(&seekReaderAt{file: file.File}, start, size)
}

func sendRaw(c fiber.Ctx, file application.RawStream) error {
	c.Set(fiber.HeaderAcceptRanges, "bytes")
	if c.Get(fiber.HeaderRange) == "" || c.Get(fiber.HeaderIfRange) != "" || file.Size == 0 {
		return c.SendStream(sectionStream{
			SectionReader: newSection(file, 0, file.Size), Closer: file.File,
		}, int(file.Size))
	}

	r, err := c.Range(file.Size)
	if errors.Is(err, fiber.ErrRequestedRangeNotSatisfiable) {
		c.Status(fiber.StatusRequestedRangeNotSatisfiable)
		c.Set(fiber.HeaderContentRange, "bytes */"+strconv.FormatInt(file.Size, 10))
		_ = file.File.Close()
		return c.Send(nil)
	}
	if err != nil || len(r.Ranges) != 1 {
		c.Status(fiber.StatusOK)
		c.Response().Header.Del(fiber.HeaderContentRange)
		return c.SendStream(sectionStream{
			SectionReader: newSection(file, 0, file.Size), Closer: file.File,
		}, int(file.Size))
	}

	part := r.Ranges[0]
	size := part.End - part.Start + 1
	c.Status(fiber.StatusPartialContent)
	c.Set(fiber.HeaderContentRange, "bytes "+strconv.FormatInt(part.Start, 10)+"-"+
		strconv.FormatInt(part.End, 10)+"/"+strconv.FormatInt(file.Size, 10))
	return c.SendStream(sectionStream{
		SectionReader: newSection(file, part.Start, size), Closer: file.File,
	}, int(size))
}
