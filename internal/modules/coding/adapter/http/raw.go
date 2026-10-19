package http

import (
	"mime"
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// fileType is how a file is served: its media type, and whether a browser may show it in the page. Anything not
// listed is a download of unknown bytes, so a project file can never run as a page of this site.
type fileType struct {
	media  string
	inline bool
}

var fileTypes = map[string]fileType{
	".pdf":  {"application/pdf", true},
	".png":  {"image/png", true},
	".jpg":  {"image/jpeg", true},
	".jpeg": {"image/jpeg", true},
	".gif":  {"image/gif", true},
	".webp": {"image/webp", true},
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", false},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", false},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", false},
	".zip":  {"application/zip", false},
}

// raw sends one file as it is: shown in the page for a PDF or a picture, otherwise as a download. download=1
// always downloads.
func (h *Handler) raw(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	file, err := workspace.Reader.ReadRaw(c.Query("path"), domain.MaxDownloadBytes)
	if err != nil {
		return fail(c, err)
	}

	kind, known := fileTypes[strings.ToLower(path.Ext(file.Path))]
	if !known {
		kind = fileType{media: "application/octet-stream"}
	}

	disposition := "attachment"
	if kind.inline && c.Query("download") != "1" {
		disposition = "inline"
	}

	c.Set("Content-Type", kind.media)
	c.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": path.Base(file.Path)}))
	c.Set("X-Content-Type-Options", "nosniff")
	// A browser's PDF viewer does not start inside a sandbox, so a file shown in the page may only be framed by this
	// site; a download is sandboxed in case it is opened anyway.
	if disposition == "inline" {
		c.Set("Content-Security-Policy", "frame-ancestors 'self'")
	} else {
		c.Set("Content-Security-Policy", "sandbox")
	}

	c.Set("Cache-Control", "private, no-store")

	return c.Send(file.Data)
}
