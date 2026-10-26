package domain

import (
	"bytes"
	"errors"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

var (
	ErrUnsupported = errors.New("unsupported upload type")
	ErrEmpty       = errors.New("upload is empty")
)

type Upload struct {
	Kind  string
	Ext   string
	Media string
	Limit int64
}

// Check validates content while it is streamed into storage.
type Check interface {
	io.Writer
	Err() error
}

// Classify detects an allowed format from its bytes, using the name only to
// decide whether otherwise-unrecognizable text is an allowed text extension.
func Classify(given string, head []byte) (Upload, Check, error) {
	if len(head) == 0 {
		return Upload{}, nil, ErrEmpty
	}

	if strings.EqualFold(extension(given), ".zip") && bytes.HasPrefix(head, []byte("PK\x03\x04")) {
		return Upload{Kind: "archive", Ext: ".zip", Media: "application/zip", Limit: MaxUploadBytes},
			newContentCheck("archive", ".zip"), nil
	}

	if upload, ok := classifyBinary(head); ok {
		return upload, newContentCheck(upload.Kind, upload.Ext), nil
	}

	if allowedTextName(given) && validTextPrefix(head) {
		ext := strings.ToLower(extension(given))
		media := map[string]string{
			".txt": "text/plain; charset=utf-8", ".md": "text/markdown; charset=utf-8",
			".csv": "text/csv; charset=utf-8", ".json": "application/json", ".log": "text/plain; charset=utf-8",
		}[ext]
		upload := Upload{Kind: "text", Ext: ext, Media: media, Limit: MaxUploadBytes}
		return upload, newContentCheck(upload.Kind, upload.Ext), nil
	}

	return Upload{}, nil, ErrUnsupported
}

func allowedTextName(name string) bool {
	switch strings.ToLower(extension(name)) {
	case ".txt", ".md", ".csv", ".json", ".log":
		return true
	default:
		return false
	}
}

func extension(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	return path.Ext(name)
}

func validTextPrefix(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	for offset := 0; offset < len(data); {
		if !utf8.FullRune(data[offset:]) {
			return offset+3 >= len(data)
		}
		_, width := utf8.DecodeRune(data[offset:])
		if width == 1 && data[offset] >= utf8.RuneSelf {
			return false
		}
		offset += width
	}
	return true
}
