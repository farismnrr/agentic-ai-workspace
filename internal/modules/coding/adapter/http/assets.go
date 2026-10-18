package http

import (
	"bytes"
	"io"
	nethttp "net/http"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	assetsFolder = "assets/"
	svgSniffSize = 1024 // how much of the start of a file is searched for "<svg"
	maxNameRunes = 40
)

// pictureTypes are the kinds of picture a person may add, found by looking at the bytes, never the file name.
var pictureTypes = map[string]string{
	"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/gif": ".gif",
}

// pictureExt returns the extension to save the data under, or "" when it is not a picture we accept.
func pictureExt(data []byte) string {
	if ext, ok := pictureTypes[nethttp.DetectContentType(data)]; ok {
		return ext
	}

	head := bytes.ToLower(data[:min(len(data), svgSniffSize)])
	if bytes.Contains(head, []byte("<svg")) && !bytes.Contains(bytes.ToLower(data), []byte("<script")) {
		return ".svg"
	}

	return ""
}

// assetName turns the uploaded name into a short safe file name under assets/, with the extension we found.
func assetName(given, ext string) string {
	base := strings.TrimSuffix(path.Base(strings.ReplaceAll(given, "\\", "/")), path.Ext(given))

	var kept strings.Builder

	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			kept.WriteRune(r)
		case r == '-' || r == '_' || r == ' ' || r == '.':
			kept.WriteByte('-')
		}
	}

	name := strings.Trim(kept.String(), "-")
	if name == "" {
		name = "picture"
	}

	return assetsFolder + name[:min(len(name), maxNameRunes)] + ext
}

// upload saves one picture the person attached into the project's assets/ folder. It never replaces a file.
func (h *Handler) upload(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	header, err := c.FormFile("file")
	if err != nil || header.Size > domain.MaxAssetBytes {
		return fail(c, domain.ErrTooLarge)
	}

	file, err := header.Open()
	if err != nil {
		return fail(c, domain.ErrIsDir)
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, domain.MaxAssetBytes+1))
	if err != nil {
		return fail(c, domain.ErrIsDir)
	}

	ext := pictureExt(data)
	if ext == "" {
		return fail(c, domain.ErrBinary)
	}

	target := assetName(header.Filename, ext)
	if _, err = workspace.Editor.WriteAsset(target, data); err != nil {
		return fail(c, err)
	}

	h.changes.Announce(c.Params("id"))

	return c.Status(fiber.StatusCreated).JSON(map[string]string{"path": target})
}
