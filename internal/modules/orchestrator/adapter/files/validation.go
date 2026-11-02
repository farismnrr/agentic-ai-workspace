package files

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

func validateBinary(file *os.File, s Session) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	header := make([]byte, 512)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}
	header = header[:n]
	ext := strings.ToLower(path.Ext(s.Name))
	media := http.DetectContentType(header)
	switch ext {
	case ".pdf":
		if !bytes.HasPrefix(header, []byte("%PDF-")) {
			return errors.New("invalid PDF")
		}
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		if !strings.HasPrefix(media, "image/") {
			return errors.New("invalid image")
		}
	case ".zip", ".docx", ".xlsx", ".pptx":
		if !bytes.HasPrefix(header, []byte("PK")) {
			return errors.New("invalid ZIP-based file")
		}
		return validateZIP(file, s.Size)
	}
	return nil
}

func validateZIP(file *os.File, size int64) error {
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}
	if len(reader.File) > 10000 {
		return errors.New("too many ZIP entries")
	}
	var inflated uint64
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		if strings.HasPrefix(name, "/") || name == ".." ||
			strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
			return errors.New("unsafe ZIP path")
		}
		inflated += entry.UncompressedSize64
		if inflated > 4<<30 {
			return errors.New("ZIP expanded content too large")
		}
	}
	return nil
}
