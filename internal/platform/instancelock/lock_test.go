//go:build unix

package instancelock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestOnlyOneHolderAtATimeAndTheLockIsReleasable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")

	first, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = Acquire(path); !errors.Is(err, ErrHeld) {
		t.Fatalf("a second holder must be refused, got %v", err)
	}

	first()

	second, err := Acquire(path)
	if err != nil {
		t.Fatalf("after release: %v", err)
	}

	second()
}

func TestAnUnusablePathIsAnError(t *testing.T) {
	if _, err := Acquire(filepath.Join(t.TempDir(), "missing", "x.lock")); err == nil || errors.Is(err, ErrHeld) {
		t.Fatalf("got %v", err)
	}
}
