package folder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const (
	dirName = "projects"
	dirMode = 0o750
)

var (
	idPattern = regexp.MustCompile(`^[a-f0-9]{8,64}$`)
	errBadID  = errors.New("invalid project id")
)

// Folders creates one folder per project under DATA_DIR/projects.
type Folders struct {
	root *os.Root
	base string
}

// Open prepares the projects folder.
func Open(dataDir string) (*Folders, error) {
	base, err := filepath.Abs(filepath.Join(dataDir, dirName))
	if err != nil {
		return nil, fmt.Errorf("resolve projects dir: %w", err)
	}

	if err = os.MkdirAll(base, dirMode); err != nil {
		return nil, fmt.Errorf("create projects dir: %w", err)
	}

	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, fmt.Errorf("open projects dir: %w", err)
	}

	return &Folders{root: root, base: base}, nil
}

// Create makes the folder for a project id.
func (f *Folders) Create(id string) error {
	if !idPattern.MatchString(id) {
		return errBadID
	}

	if err := f.root.Mkdir(id, dirMode); err != nil {
		return fmt.Errorf("create project folder: %w", err)
	}

	return nil
}

// Dir is the absolute path of a project's folder.
func (f *Folders) Dir(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", errBadID
	}

	return filepath.Join(f.base, id), nil
}
