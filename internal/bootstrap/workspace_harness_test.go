package bootstrap

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// testImage is small and has the shell tools the tests need.
const testImage = "busybox:1.37"

var (
	dockerOnce sync.Once
	errDocker  error
)

// requireDocker skips a test that runs a task when the host has no usable Docker.
func requireDocker(t *testing.T) {
	t.Helper()

	dockerOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		_, errDocker = sandbox.Open(ctx, sandbox.Config{Image: testImage, Limits: sandbox.DefaultLimits()})
	})

	if errDocker != nil {
		t.Skipf("no Docker on this host: %v", errDocker)
	}
}

// testStore is what a test needs to reach the app's data without going through its API.
type testStore struct {
	db      *sql.DB
	dataDir string
}

// testStores remembers the store behind each test app.
var testStores sync.Map

var workspaceFixtures = map[string]string{
	"hello.txt":             "hello from the workspace\n",
	".workspace/tasks.json": `{"tasks":{"echo":{"description":"Say hello","command":["echo","hello task"]}}}`,
}

// userWorkspace is the folder of the project that the app named client works in for the person
// behind cookie. The project is made if the app has not called yet, as its first call would.
func userWorkspace(t *testing.T, app *fiber.App, cookie, client string) string {
	t.Helper()

	me := do(t, app, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)
	userID, _ := me["id"].(string)
	value, _ := testStores.Load(app)

	store, _ := value.(testStore)
	if userID == "" || store.db == nil {
		t.Fatalf("cannot locate the workspace of %v", me)
	}

	var id string

	ctx := context.Background()
	err := store.db.QueryRowContext(ctx,
		`SELECT id FROM projects WHERE user_id = ? AND source = ? AND archived_at IS NULL`, userID, client).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		id = newTestID(t)
		_, err = store.db.ExecContext(ctx, `INSERT INTO projects (id, user_id, name, description, source, created_at)
			VALUES (?, ?, ?, '', ?, ?)`, id, userID, client, client, time.Now().Unix())
	}

	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(store.dataDir, "projects", id)
	if err = os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}

	return dir
}

func newTestID(t *testing.T) string {
	t.Helper()

	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}

	return hex.EncodeToString(raw)
}

// seedWorkspace gives the project of the app named client the sample files the tests read.
func seedWorkspace(t *testing.T, app *fiber.App, cookie, client string) {
	t.Helper()

	dir := userWorkspace(t, app, cookie, client)

	for name, body := range workspaceFixtures {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
