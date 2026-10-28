package apptest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var workspaceFixtures = map[string]string{
	"hello.txt":             "hello from the workspace\n",
	".workspace/tasks.json": `{"tasks":{"echo":{"description":"Say hello","command":["echo","hello task"]}}}`,
}

// UserWorkspace returns the project folder associated with an external client.
func UserWorkspace(t *testing.T, env Environment, cookie, client string) string {
	t.Helper()

	me := Do(t, env.App, "GET", "/api/auth/me", "", map[string]string{"Cookie": cookie}).JSON(t)
	userID, _ := me["id"].(string)
	if userID == "" || env.DB == nil {
		t.Fatalf("cannot locate the workspace of %v", me)
	}

	var id string
	err := env.DB.QueryRowContext(context.Background(),
		`SELECT id FROM projects WHERE user_id = ? AND source = ? AND archived_at IS NULL`, userID, client).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		id = workspaceID(t)
		_, err = env.DB.ExecContext(context.Background(), `INSERT INTO projects
			(id, user_id, name, description, slug, source, created_at)
			VALUES (?, ?, ?, '', ?, ?, ?)`, id, userID, client, "app-"+id, client, time.Now().Unix())
	}
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(env.DataDir, "projects", id)
	if err = os.MkdirAll(dir, testDirectoryMode); err != nil {
		t.Fatal(err)
	}
	return dir
}

// SeedWorkspace writes the standard fixture files into the external client's project.
func SeedWorkspace(t *testing.T, env Environment, cookie, client string) {
	t.Helper()

	root := UserWorkspace(t, env, cookie, client)
	for name, body := range workspaceFixtures {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), testDirectoryMode); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), testFileMode); err != nil {
			t.Fatal(err)
		}
	}
}

func workspaceID(t *testing.T) string {
	t.Helper()

	raw := make([]byte, randomIDBytes)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(raw)
}
