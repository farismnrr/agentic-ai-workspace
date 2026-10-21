package http_test

import (
	"bytes"
	"context"
	"mime/multipart"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/backup"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	codinghttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type ids struct{ n int }

func (i *ids) ID() (string, error) {
	i.n++

	return "abcdef0" + strconv.Itoa(i.n) + "00", nil
}

type owner struct {
	workspace codinghttp.Workspace
	stranger  bool
}

func (o owner) Open(context.Context, string, string) (codinghttp.Workspace, error) {
	if o.stranger {
		return codinghttp.Workspace{}, domain.ErrNotFound
	}

	return o.workspace, nil
}

type quiet struct{}

func (quiet) Announce(string) {}
func (quiet) Follow(string) (<-chan struct{}, func()) {
	return make(chan struct{}), func() {}
}

func newApp(t *testing.T) (*fiber.App, string) {
	t.Helper()

	root := t.TempDir()

	files, err := fs.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	backups, err := backup.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	workspace := codinghttp.Workspace{
		Reader: application.NewService(files),
		Editor: application.NewEditor(files, files, backups, &ids{}),
	}
	pass := func(c fiber.Ctx) error { return c.Next() }
	app := fiber.New(fiber.Config{BodyLimit: int(domain.MaxVideoBytes + (1 << 20))})
	codinghttp.NewHandler(owner{workspace: workspace}, quiet{}).Mount(pass, pass, nil)(app)

	return app, root
}

func upload(t *testing.T, app *fiber.App, name string, data []byte) int {
	t.Helper()

	var body bytes.Buffer

	form := multipart.NewWriter(&body)

	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = part.Write(data)
	_ = form.Close()

	req := httptest.NewRequestWithContext(context.Background(), nethttp.MethodPost, "/api/projects/p/assets", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	return res.StatusCode
}

var png = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func TestAPictureIsSavedUnderAssetsAndNeverReplaced(t *testing.T) {
	t.Parallel()

	app, root := newApp(t)

	if got := upload(t, app, "My Hero Photo!.PNG", png); got != fiber.StatusCreated {
		t.Fatalf("first upload = %d", got)
	}

	if _, err := os.Stat(filepath.Join(root, "assets", "my-hero-photo.png")); err != nil {
		t.Fatalf("saved file: %v", err)
	}

	if got := upload(t, app, "My Hero Photo!.PNG", png); got != fiber.StatusConflict {
		t.Fatalf("second upload = %d", got)
	}
}

func TestOnlyPicturesAreAccepted(t *testing.T) {
	t.Parallel()

	app, _ := newApp(t)

	for name, data := range map[string][]byte{
		"run.png":  []byte("#!/bin/sh\necho hi\n"),
		"a.svg":    []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"../x.png": []byte("not a picture"),
	} {
		if got := upload(t, app, name, data); got != fiber.StatusUnsupportedMediaType {
			t.Errorf("%s = %d", name, got)
		}
	}

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="1"/></svg>`)
	if got := upload(t, app, "logo.svg", svg); got != fiber.StatusCreated {
		t.Errorf("svg = %d", got)
	}
}

func TestAnotherPersonsProjectIsNotFound(t *testing.T) {
	t.Parallel()
	app := strangerApp()

	if got := upload(t, app, "a.png", png); got != fiber.StatusNotFound {
		t.Fatalf("upload to another person's project = %d", got)
	}
}

func strangerApp() *fiber.App {
	app := fiber.New()
	pass := func(c fiber.Ctx) error { return c.Next() }
	codinghttp.NewHandler(owner{stranger: true}, quiet{}).Mount(pass, pass, nil)(app)
	return app
}
