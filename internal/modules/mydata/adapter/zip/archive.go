// Package zip writes exports as a zip file.
package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"time"
)

// Archive writes files into a zip stream.
type Archive struct {
	writer *zip.Writer
	now    time.Time
}

// New starts a zip on w; files are stamped with now.
func New(w io.Writer, now time.Time) *Archive {
	return &Archive{writer: zip.NewWriter(w), now: now}
}

// Create starts a compressed file at path.
func (a *Archive) Create(path string) (io.Writer, error) {
	file, err := a.writer.CreateHeader(&zip.FileHeader{Name: path, Method: zip.Deflate, Modified: a.now})
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}

	return file, nil
}

// Close finishes the zip.
func (a *Archive) Close() error {
	if err := a.writer.Close(); err != nil {
		return fmt.Errorf("finish zip: %w", err)
	}

	return nil
}
