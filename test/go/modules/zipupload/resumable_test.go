package zipupload_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	coding "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Quota tests must not depend on the CI runner's actual free disk space.
// Production still uses fs.Root's real available-capacity check.
type testRoot struct{ *fs.Root }

func (r *testRoot) AvailableUploadBytes() (int64, error) { return 16 << 30, nil }

func editor(t *testing.T) (*coding.Editor, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := fs.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return coding.NewEditor(root, &testRoot{Root: root}, nil, nil), dir
}

func makeZIP(t *testing.T, entry string) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	header := &zip.FileHeader{Name: entry, Method: zip.Store}
	contained, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(contained, "hello secure ZIP"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestResumableZIPReopensFromDurableOffsetAndPublishesOnce(t *testing.T) {
	app, dir := editor(t)
	raw := makeZIP(t, "app/index.txt")
	session, err := app.StartZIP("owner", "my-project.zip", int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ZIPStatus(session.ID, "other-user"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("session read must be owner-scoped: %v", err)
	}
	split := len(raw) / 2
	part, err := app.AppendZIP(session.ID, "owner", 0, bytes.NewReader(raw[:split]), int64(split))
	if err != nil || part.Offset != int64(split) {
		t.Fatalf("append: %v %+v", err, part)
	}
	if _, err := app.AppendZIP(session.ID, "owner", 0, bytes.NewReader(raw[:2]), 2); !errors.Is(err, coding.ErrZIPConflict) {
		t.Fatalf("replay must fail closed: %v", err)
	}
	resumed, err := app.ZIPStatus(session.ID, "owner")
	if err != nil || resumed.Offset != int64(split) {
		t.Fatalf("offset lost: %v %+v", err, resumed)
	}
	complete, err := app.AppendZIP(session.ID, "owner", resumed.Offset,
		bytes.NewReader(raw[split:]), int64(len(raw)-split))
	if err != nil || complete.Offset != int64(len(raw)) {
		t.Fatalf("append last: %v %+v", err, complete)
	}
	if complete.State != "validating" {
		t.Fatalf("complete transfer must transition to validating: %+v", complete)
	}
	if _, err := app.StartZIP("owner", "overlapping.zip", int64(len(raw))); !errors.Is(err, coding.ErrZIPQuota) {
		t.Fatalf("validating ZIP must still reserve quota and project upload slot: %v", err)
	}
	ready, err := app.FinishZIP(session.ID, "owner")
	if err != nil || ready.State != "ready" || ready.Path == "" {
		t.Fatalf("publish: %v %+v", err, ready)
	}
	replayed, err := app.ZIPStatus(session.ID, "owner")
	if err != nil || replayed.Path != ready.Path {
		t.Fatalf("idempotent status: %v %+v", err, replayed)
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, ready.Path))
	if err != nil || !bytes.Equal(onDisk, raw) {
		t.Fatalf("published ZIP differs from original: %v", err)
	}
	sum := sha256.Sum256(raw)
	if ready.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("expected SHA256, got %s", ready.SHA256)
	}
	if _, err := os.Stat(filepath.Join(dir, ".workspace", "resumable-uploads", session.ID+".part")); !os.IsNotExist(err) {
		t.Fatalf("staging must be moved away: %v", err)
	}
}

func TestResumableZIPRejectsUnsafeArchiveAndReleasesReservation(t *testing.T) {
	for _, entry := range []string{"../secret.txt", "/etc/passwd", "foo/../../secret"} {
		t.Run(entry, func(t *testing.T) {
			app, dir := editor(t)
			raw := makeZIP(t, entry)
			session, err := app.StartZIP("owner", "repo.zip", int64(len(raw)))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := app.AppendZIP(session.ID, "owner", 0, bytes.NewReader(raw), int64(len(raw))); err != nil {
				t.Fatal(err)
			}
			if _, err := app.FinishZIP(session.ID, "owner"); err == nil {
				t.Fatal("unsafe ZIP was published")
			}
			if _, err := os.Stat(filepath.Join(dir, "uploads")); err == nil {
				entries, listErr := os.ReadDir(filepath.Join(dir, "uploads"))
				if listErr != nil || len(entries) > 0 {
					t.Fatalf("rejected ZIP was published: %v", entries)
				}
			}
			if _, err := app.StartZIP("owner", "retry.zip", int64(len(raw))); err != nil {
				t.Fatalf("rejected ZIP held quota reservation: %v", err)
			}
		})
	}
}

func TestResumableZIPBoundsAndCancel(t *testing.T) {
	app, _ := editor(t)
	if _, err := app.StartZIP("owner", "huge.zip", domain.MaxZIPBytes+1); !errors.Is(err, domain.ErrTooLarge) {
		t.Fatalf("oversized ZIP should fail: %v", err)
	}
	session, err := app.StartZIP("owner", "fifty.zip", 50<<20)
	if err != nil {
		t.Fatalf("50MiB ZIP should be allowed: %v", err)
	}
	if _, err := app.StartZIP("owner", "second.zip", 50<<20); !errors.Is(err, coding.ErrZIPQuota) {
		t.Fatalf("parallel project reservation should be blocked: %v", err)
	}
	if err := app.CancelZIP(session.ID, "wrong-user"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user cancel must be denied: %v", err)
	}
	if err := app.CancelZIP(session.ID, "owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.StartZIP("owner", "second.zip", 50<<20); err != nil {
		t.Fatalf("cancel failed to release reservation: %v", err)
	}
}
