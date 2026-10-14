package backup

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"
)

const (
	dirName  = "backups"
	dirMode  = 0o700
	fileMode = 0o600
	maxAge   = 30 * 24 * time.Hour
)

var errBadID = errors.New("invalid backup id")

var idPattern = regexp.MustCompile(`^[a-f0-9]{8,64}$`)

// Store keeps previous file contents under DATA_DIR/backups, outside the
// workspace, so a change can be undone even if the workspace is damaged.
type Store struct {
	root *os.Root
}

// Open prepares the backup folder and drops backups older than 30 days.
func Open(dataDir string) (*Store, error) {
	dir := dataDir + string(os.PathSeparator) + dirName
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open backup dir: %w", err)
	}

	store := &Store{root: root}
	store.prune(time.Now().Add(-maxAge))

	return store, nil
}

// Save stores content under id.
func (s *Store) Save(id string, data []byte) error {
	if !idPattern.MatchString(id) {
		return errBadID
	}

	if err := s.root.WriteFile(id, data, fileMode); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}

	return nil
}

// Load returns the content saved under id.
func (s *Store) Load(id string) ([]byte, error) {
	if !idPattern.MatchString(id) {
		return nil, errBadID
	}

	data, err := s.root.ReadFile(id)
	if err != nil {
		return nil, fmt.Errorf("read backup: %w", err)
	}

	return data, nil
}

func (s *Store) prune(before time.Time) {
	dir, err := s.root.Open(".")
	if err != nil {
		return
	}

	defer func() { _ = dir.Close() }()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if info, infoErr := entry.Info(); infoErr == nil && info.ModTime().Before(before) {
			_ = s.root.Remove(entry.Name())
		}
	}
}
