package http

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const tusVersion = "1.0.0"

func (h *Handler) zipLock(project string) func() {
	actual, _ := h.zipLocks.LoadOrStore(project, &sync.Mutex{})
	lock := actual.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

func zipHeaders(c fiber.Ctx) {
	c.Set("Tus-Resumable", tusVersion)
	c.Set("Cache-Control", "no-store")
}

func zipFailure(c fiber.Ctx, err error) error {
	zipHeaders(c)
	switch {
	case errors.Is(err, application.ErrZIPConflict), errors.Is(err, application.ErrZIPState):
		return c.Status(409).JSON(httpserver.Failure("upload_conflict"))
	case errors.Is(err, application.ErrZIPExpired):
		return c.Status(410).JSON(httpserver.Failure("upload_expired"))
	case errors.Is(err, application.ErrZIPQuota):
		return c.Status(507).JSON(httpserver.Failure("upload_quota"))
	default:
		return fail(c, err)
	}
}

func tusUploadName(header string) (string, bool) {
	for _, field := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), " ")
		if !ok || key != "filename" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(value)
		if err == nil && len(raw) > 0 && len(raw) <= 255 {
			return string(raw), true
		}
	}
	return "", false
}

func validTus(c fiber.Ctx) bool {
	return c.Get("Tus-Resumable") == tusVersion
}

func (h *Handler) zipOptions(c fiber.Ctx) error {
	c.Set("Tus-Version", tusVersion)
	c.Set("Tus-Extension", "creation,termination,expiration")
	c.Set("Tus-Max-Size", strconv.FormatInt(domain.MaxZIPBytes, 10))
	c.Set("Upload-Max-Chunk-Size", strconv.FormatInt(domain.MaxZIPChunkBytes, 10))
	return c.SendStatus(204)
}

func (h *Handler) zipLimits(c fiber.Ctx) error {
	if _, err := h.open(c); err != nil {
		return zipFailure(c, err)
	}
	c.Set("Cache-Control", "no-store")
	return c.JSON(fiber.Map{"zipBytes": domain.MaxZIPBytes,
		"chunkBytes": domain.MaxZIPChunkBytes, "quotaBytes": domain.MaxProjectUploads})
}

func (h *Handler) zipCreate(c fiber.Ctx) error {
	if !validTus(c) {
		return c.Status(412).JSON(httpserver.Failure("tus_version"))
	}
	workspace, err := h.open(c)
	if err != nil {
		return zipFailure(c, err)
	}
	size, err := strconv.ParseInt(c.Get("Upload-Length"), 10, 64)
	if err != nil || size < 22 {
		return c.Status(400).JSON(httpserver.Failure("upload_length"))
	}
	name, ok := tusUploadName(c.Get("Upload-Metadata"))
	if !ok {
		return c.Status(400).JSON(httpserver.Failure("upload_filename"))
	}
	unlock := h.zipLock(c.Params("id"))
	defer unlock()
	session, err := workspace.Editor.StartZIP(httpserver.UserID(c), name, size)
	if err != nil {
		return zipFailure(c, err)
	}
	zipHeaders(c)
	c.Set("Location", "/api/projects/"+c.Params("id")+"/uploads/resumable/"+session.ID)
	c.Set("Upload-Offset", "0")
	c.Set("Upload-Expires", session.Updated.Add(24*time.Hour).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	return c.Status(201).JSON(session)
}

// ZIP validation is separate from the final chunk request. HEAD may restart
// interrupted validations after an app process restart.
func (h *Handler) validateZipLater(project string, workspace Workspace, id, owner string) {
	key := project + ":" + id
	if _, loaded := h.zipWorkers.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	go func() {
		defer h.zipWorkers.Delete(key)
		unlock := h.zipLock(project)
		defer unlock()
		ready, err := workspace.Editor.FinishZIP(id, owner)
		if err == nil && ready.State == "ready" {
			h.changes.Announce(project)
		}
	}()
}

func (h *Handler) zipHead(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return zipFailure(c, err)
	}
	unlock := h.zipLock(c.Params("id"))
	defer unlock()
	session, err := workspace.Editor.ZIPStatus(c.Params("uploadId"), httpserver.UserID(c))
	if err != nil {
		return zipFailure(c, err)
	}
	if session.State == "validating" || session.State == "publishing" ||
		(session.State == "uploading" && session.Offset == session.Size) {
		h.validateZipLater(c.Params("id"), workspace, session.ID, httpserver.UserID(c))
		session.State = "validating"
	}
	zipHeaders(c)
	c.Set("Upload-Offset", strconv.FormatInt(session.Offset, 10))
	c.Set("Upload-Length", strconv.FormatInt(session.Size, 10))
	c.Set("Upload-State", session.State)
	if session.Path != "" {
		c.Set("Upload-Path", session.Path)
	}
	return c.SendStatus(200)
}

func (h *Handler) zipPatch(c fiber.Ctx) error {
	if !validTus(c) {
		return c.Status(412).JSON(httpserver.Failure("tus_version"))
	}
	if c.Get("Content-Type") != "application/offset+octet-stream" {
		return c.Status(415).JSON(httpserver.Failure("unsupported_type"))
	}
	// Reject the chunk based on its framed size before we touch workspace data.
	if len(c.Body()) == 0 || len(c.Body()) > int(domain.MaxZIPChunkBytes) {
		return c.Status(413).JSON(httpserver.Failure("too_large"))
	}
	offset, err := strconv.ParseInt(c.Get("Upload-Offset"), 10, 64)
	if err != nil || offset < 0 {
		return c.Status(400).JSON(httpserver.Failure("upload_offset"))
	}
	workspace, err := h.open(c)
	if err != nil {
		return zipFailure(c, err)
	}
	unlock := h.zipLock(c.Params("id"))
	defer unlock()
	session, err := workspace.Editor.AppendZIP(c.Params("uploadId"), httpserver.UserID(c),
		offset, bytes.NewReader(c.Body()), int64(len(c.Body())))
	if err != nil {
		if errors.Is(err, application.ErrZIPConflict) {
			c.Set("Upload-Offset", strconv.FormatInt(session.Offset, 10))
		}
		return zipFailure(c, err)
	}
	if session.Offset == session.Size {
		h.validateZipLater(c.Params("id"), workspace, session.ID, httpserver.UserID(c))
	}
	zipHeaders(c)
	c.Set("Upload-Offset", strconv.FormatInt(session.Offset, 10))
	c.Set("Upload-State", session.State)
	code := 200
	if session.State == "validating" {
		code = 202
	}
	return c.Status(code).JSON(fiber.Map{"offset": session.Offset, "state": session.State, "path": session.Path})
}

func (h *Handler) zipDelete(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return zipFailure(c, err)
	}
	unlock := h.zipLock(c.Params("id"))
	defer unlock()
	if err := workspace.Editor.CancelZIP(c.Params("uploadId"), httpserver.UserID(c)); err != nil {
		return zipFailure(c, err)
	}
	zipHeaders(c)
	return c.SendStatus(204)
}
