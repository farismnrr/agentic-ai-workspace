package archive_test

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/adapter/archive"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/extension/domain"
)

type folders struct{ dir string }

func (f folders) Dir(string) (string, error) { return f.dir, nil }

func project(t *testing.T) (*archive.Packer, string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "dist", "sub"), 0o750); err != nil {
		t.Fatal(err)
	}

	for name, content := range map[string]string{
		"dist/manifest.json": `{"manifest_version":3}`, "dist/popup.html": "x", "dist/sub/a.js": "y",
		"dist/.DS_Store": "junk",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return archive.New(folders{dir: dir}, t.TempDir()), dir
}

func TestTheFolderIsReadForItsManifestAndFiles(t *testing.T) {
	t.Parallel()

	packer, _ := project(t)

	tree, err := packer.Inspect(context.Background(), "p1", "dist")
	if err != nil || tree.Files != 4 || !tree.Has("sub/a.js") || tree.Has("nope.js") || len(tree.Manifest) == 0 {
		t.Fatalf("tree %+v err %v", tree, err)
	}
}

func TestTheFolderIsZippedWithTheManifestAtTheRoot(t *testing.T) {
	t.Parallel()

	packer, _ := project(t)
	ctx := context.Background()

	if err := packer.Pack(ctx, "p1", "dist"); err != nil {
		t.Fatal(err)
	}

	file, size, err := packer.Open("p1")
	if err != nil || size == 0 {
		t.Fatalf("open: %v %d", err, size)
	}

	defer func() { _ = file.Close() }()

	data, _ := io.ReadAll(file)

	names := zipNames(t, data)
	if !names["manifest.json"] || !names["sub/a.js"] || names[".DS_Store"] {
		t.Fatalf("the package holds the build, without what a system leaves behind: %v", names)
	}
}

func zipNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()

	reader, err := zip.NewReader(bytesReaderAt(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, f := range reader.File {
		names[f.Name] = true
	}

	return names
}

func TestAFolderOutsideTheProjectOrWithoutAManifestIsRefused(t *testing.T) {
	t.Parallel()

	packer, dir := project(t)
	ctx := context.Background()

	for _, folder := range []string{"../etc", "/etc", "missing"} {
		if _, err := packer.Inspect(ctx, "p1", folder); !errors.Is(err, domain.ErrBadFolder) {
			t.Errorf("%q must be refused, got %v", folder, err)
		}
	}

	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o750); err != nil {
		t.Fatal(err)
	}

	if _, err := packer.Inspect(ctx, "p1", "empty"); !errors.Is(err, domain.ErrNoManifest) {
		t.Errorf("a folder with no manifest.json: %v", err)
	}
}

func TestALinkOutOfTheFolderIsNeitherFollowedNorPackaged(t *testing.T) {
	t.Parallel()

	packer, dir := project(t)

	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("password"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(secret, filepath.Join(dir, "dist", "leak.txt")); err != nil {
		t.Skip("links are not available here")
	}

	tree, err := packer.Inspect(context.Background(), "p1", "dist")
	if err != nil || tree.Has("leak.txt") {
		t.Fatalf("a link is not a file of the extension: %+v %v", tree, err)
	}
}

func TestAMissingPackageIsNotFound(t *testing.T) {
	t.Parallel()

	packer, _ := project(t)
	if _, _, err := packer.Open("never"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v", err)
	}

	if err := packer.Remove("never"); err != nil {
		t.Fatalf("removing nothing is fine: %v", err)
	}
}

type memory []byte

func (m memory) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(m)) {
		return 0, io.EOF
	}

	n := copy(p, m[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func bytesReaderAt(data []byte) io.ReaderAt { return memory(data) }
