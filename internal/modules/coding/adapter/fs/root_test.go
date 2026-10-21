package fs_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/fs"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func put(t *testing.T, root, name, body string) {
	t.Helper()

	full := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func workspace(t *testing.T) *application.Service {
	t.Helper()

	root, outside := t.TempDir(), t.TempDir()

	put(t, root, "main.go", "package main\n\nfunc Hello() {}\n")
	put(t, root, "src/util.go", "package src\n// hello helper\n")
	put(t, root, ".env", "SECRET=1")
	put(t, root, ".git/config", "[core]")
	put(t, root, "node_modules/dep/index.js", "hello")
	put(t, root, "blob.bin", "abc\x00def")
	put(t, outside, "secret.txt", "top secret")

	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	files, err := fs.Open(root)
	if err != nil {
		t.Fatal(err)
	}

	return application.NewService(files)
}

func paths(listing application.Listing) map[string]bool {
	found := map[string]bool{}
	for _, entry := range listing.Entries {
		found[entry.Path] = true
	}

	return found
}

func TestListHidesSecrets(t *testing.T) {
	t.Parallel()

	listing, err := workspace(t).List("")
	if err != nil {
		t.Fatal(err)
	}

	found := paths(listing)
	if found[".env"] || found[".git"] || !found["main.go"] {
		t.Fatalf("secrets must be hidden and files shown: %v", found)
	}
}

func TestListPutsDirectoriesFirst(t *testing.T) {
	t.Parallel()

	listing, err := workspace(t).List("")
	if err != nil {
		t.Fatal(err)
	}

	if first := listing.Entries[0]; !first.Dir {
		t.Fatalf("directories should come first, got %+v", first)
	}
}

func TestReadFindsContent(t *testing.T) {
	t.Parallel()

	file, err := workspace(t).Read("src/util.go")
	if err != nil || file.Content != "package src\n// hello helper\n" {
		t.Fatalf("normal read failed: %q %v", file.Content, err)
	}
}

func TestOpenReturnsConfinedSeekableFileAndRejectsDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root, err := fs.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	put(t, dir, "clip.mp4", "video")
	file, size, err := root.Open("clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if size != 5 {
		t.Fatalf("size = %d", size)
	}
	if _, err := io.ReadAll(file); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.Open("."); !errors.Is(err, domain.ErrIsDir) {
		t.Fatalf("directory open error = %v", err)
	}
}

func TestReadRefusals(t *testing.T) {
	t.Parallel()

	service := workspace(t)

	cases := map[string]error{
		"escape/secret.txt": domain.ErrDenied,
		".env":              domain.ErrDenied,
		".git/config":       domain.ErrDenied,
		"blob.bin":          domain.ErrBinary,
		"src":               domain.ErrIsDir,
		"missing.txt":       domain.ErrNotFound,
	}

	for path, want := range cases {
		if _, err := service.Read(path); !errors.Is(err, want) {
			t.Errorf("Read(%q) = %v, want %v", path, err, want)
		}
	}
}

func TestParentTraversalStaysInsideRoot(t *testing.T) {
	t.Parallel()

	if _, err := workspace(t).Read("../../etc/passwd"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("traversal must resolve inside the root and miss, got %v", err)
	}
}

func TestListingAFileExplainsItself(t *testing.T) {
	t.Parallel()

	if _, err := workspace(t).List("main.go"); !errors.Is(err, domain.ErrNotDir) {
		t.Fatalf("got %v", err)
	}
}

func TestSearchSkipsVendorSecretsAndSymlinks(t *testing.T) {
	t.Parallel()

	result, err := workspace(t).Search("HELLO", "", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Matches) != 2 {
		t.Fatalf("want hits in main.go and src/util.go only, got %+v", result.Matches)
	}
}

func TestSearchDoesNotFollowSymlinksOut(t *testing.T) {
	t.Parallel()

	result, err := workspace(t).Search("top secret", "", false)
	if err != nil || len(result.Matches) != 0 {
		t.Fatalf("symlink escape must not be searched: %+v %v", result.Matches, err)
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	t.Parallel()

	if _, err := workspace(t).Search("", "", false); !errors.Is(err, domain.ErrEmptyQuery) {
		t.Fatalf("got %v", err)
	}
}
