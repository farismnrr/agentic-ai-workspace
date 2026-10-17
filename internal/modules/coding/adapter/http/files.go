package http

import (
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type entryPayload struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Dir  bool   `json:"dir"`
}

type listPayload struct {
	Path      string         `json:"path"`
	Entries   []entryPayload `json:"entries"`
	Truncated bool           `json:"truncated"`
}

// list answers with the entries of one folder, folders first; the files the agent may not see are left out.
func (h *Handler) list(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	listing, err := workspace.Reader.List(c.Query("path", "."))
	if err != nil {
		return fail(c, err)
	}

	where := c.Query("path", ".")
	out := listPayload{Path: where, Entries: make([]entryPayload, 0, len(listing.Entries)), Truncated: listing.Truncated}

	for _, entry := range listing.Entries {
		out.Entries = append(out.Entries, entryPayload{
			Name: path.Base(entry.Path), Path: entry.Path, Size: entry.Size, Dir: entry.Dir,
		})
	}

	return c.JSON(out)
}

type filePayload struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Hash      string `json:"hash"`
	Truncated bool   `json:"truncated"`
	Binary    bool   `json:"binary"`
}

// read answers with a text file and the hash an edit must quote. A binary file or one too large to edit is shown
// as such, so the page can say so instead of offering to edit it.
func (h *Handler) read(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	file, err := workspace.Reader.Read(c.Query("path"))
	if err != nil {
		if isBinary(err) {
			return c.JSON(filePayload{Path: c.Query("path"), Binary: true})
		}

		return fail(c, err)
	}

	return c.JSON(filePayload{
		Path: file.Path, Content: file.Content, Hash: domain.ContentHash(file.Content), Truncated: file.Truncated,
	})
}

type saveInput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	BaseHash string `json:"baseHash"`
}

// save writes the person's edit, unless the file changed after they opened it (then 409 with the current hash).
func (h *Handler) save(c fiber.Ctx) error {
	var in saveInput
	if err := c.Bind().JSON(&in); err != nil || strings.TrimSpace(in.Path) == "" {
		return fail(c, domain.ErrIsDir)
	}

	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	if _, err = workspace.Editor.WriteIfUnchanged(in.Path, in.Content, in.BaseHash); err != nil {
		return fail(c, err)
	}

	h.changes.Announce(c.Params("id"))

	return c.JSON(map[string]string{"hash": domain.ContentHash(in.Content)})
}

// remove deletes one file, keeping a backup like any change.
func (h *Handler) remove(c fiber.Ctx) error {
	workspace, err := h.open(c)
	if err != nil {
		return fail(c, err)
	}

	if _, err = workspace.Editor.Delete(c.Query("path")); err != nil {
		return fail(c, err)
	}

	h.changes.Announce(c.Params("id"))

	return c.SendStatus(fiber.StatusNoContent)
}
