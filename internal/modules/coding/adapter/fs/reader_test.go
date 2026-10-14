package fs_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestFindMatchesGlobsAtAnyDepth(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	put(t, e.root, "src/a.go", "x")
	put(t, e.root, "src/deep/b.go", "x")
	put(t, e.root, "main.go", "x")

	found, err := e.reader.Find("**/*.go", "")
	if err != nil || len(found.Paths) != 3 {
		t.Fatalf("want 3 go files, got %v %v", found.Paths, err)
	}

	if only, _ := e.reader.Find("src/*.go", ""); len(only.Paths) != 1 {
		t.Fatalf("single-level glob should find 1, got %v", only.Paths)
	}
}

func TestReadLinesReturnsTheRequestedRange(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	put(t, e.root, "l.txt", "1\n2\n3\n4\n")

	file, err := e.reader.ReadLines("l.txt", 2, 3)
	if err != nil || file.Content != "2\n3" {
		t.Fatalf("got %q %v", file.Content, err)
	}

	if _, err := e.reader.ReadLines("l.txt", 3, 2); !errors.Is(err, domain.ErrBadRange) {
		t.Fatalf("inverted range must be refused, got %v", err)
	}
}
