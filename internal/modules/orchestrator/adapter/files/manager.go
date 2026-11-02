// Package files persists bounded resumable artifacts for external Orchestrator Space.
// It has its own transport and state; existing Coding upload endpoints are untouched.
package files

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	orchestratorDomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/orchestrator/domain"

	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
)

const (
	stage      = ".workspace/orchestrator-uploads"
	maxChunk   = 512 << 10
	maxBinary  = 512 << 20
	maxProject = 4 << 30
	sessionTTL = 24 * time.Hour
)

type Folders interface {
	Dir(projectID string) (string, error)
}

type Manager struct {
	Projects *projectapp.Service
	Folders  Folders
	Notify   func(projectID string)
	mu       sync.Mutex
}

type Session = orchestratorDomain.UploadSession

func (m *Manager) open(ctx context.Context, userID, projectID string) (*os.Root, error) {
	if projectID == "" || userID == "" {
		return nil, errors.New("choose a project")
	}
	if _, err := m.Projects.Get(ctx, userID, projectID); err != nil {
		return nil, errors.New("project not found")
	}
	dir, err := m.Folders.Dir(projectID)
	if err != nil {
		return nil, err
	}
	return os.OpenRoot(dir)
}

func initialize(root *os.Root) error {
	for _, name := range []string{".workspace", stage, "uploads"} {
		if err := root.Mkdir(name, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create upload directory: %w", err)
		}
	}
	return validateUploadDirs(root)
}

// Reject project-controlled symlinks posing as upload directories. In particular,
// uploading through an uploads -> .git link must never mutate protected paths.
func validateUploadDirs(root *os.Root) error {
	for _, name := range []string{".workspace", stage, "uploads"} {
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("unsafe upload directory")
		}
	}
	return nil
}

func metaPath(id string) string { return path.Join(stage, id+".json") }
func partPath(id string) string { return path.Join(stage, id+".part") }

func (m *Manager) read(root *os.Root, userID, projectID, id string) (Session, error) {
	if err := validateUploadDirs(root); err != nil {
		return Session{}, err
	}
	if len(id) != 32 {
		return Session{}, errors.New("invalid upload ID")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return Session{}, errors.New("invalid upload ID")
	}
	data, err := root.ReadFile(metaPath(id))
	if err != nil {
		return Session{}, errors.New("upload session not found")
	}
	var s Session
	if json.Unmarshal(data, &s) != nil || s.ID != id || s.Owner != userID || s.Project != projectID {
		return Session{}, errors.New("upload session not found")
	}
	if time.Now().After(s.Expires) && s.Status != "done" {
		return Session{}, errors.New("upload session expired")
	}
	if s.Status == "uploading" {
		fi, err := root.Stat(partPath(id))
		if err != nil {
			return Session{}, errors.New("upload data missing")
		}
		if fi.Size() > s.Size {
			return Session{}, errors.New("corrupted upload")
		}
		s.Offset = fi.Size()
	}
	return s, nil
}

func save(root *os.Root, s Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	next := metaPath(s.ID) + ".next"
	if err := root.WriteFile(next, data, 0o600); err != nil {
		return err
	}
	return root.Rename(next, metaPath(s.ID))
}

func validName(name string) bool {
	if len(name) < 3 || len(name) > 128 || strings.HasPrefix(name, ".") ||
		strings.ContainsAny(name, "/\\") || name == ".." {
		return false
	}
	switch strings.ToLower(path.Ext(name)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp",
		".zip", ".docx", ".xlsx", ".pptx", ".mp4", ".webm", ".csv", ".txt":
		return true
	}
	return false
}

func (m *Manager) Start(ctx context.Context, userID, projectID, taskID, name string, size int64, hash string) (Session, error) {
	if !validName(name) || size <= 0 || size > maxBinary || len(hash) != 64 {
		return Session{}, errors.New("invalid filename, size or SHA-256")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return Session{}, errors.New("invalid SHA-256")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	root, err := m.open(ctx, userID, projectID)
	if err != nil {
		return Session{}, err
	}
	defer root.Close()
	if err := initialize(root); err != nil {
		return Session{}, err
	}
	if _, err := root.Stat(path.Join("uploads", name)); err == nil {
		return Session{}, errors.New("file already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Session{}, err
	}
	used, err := projectUsage(root)
	if err != nil {
		return Session{}, err
	}
	if used+size > maxProject {
		return Session{}, errors.New("project upload quota exceeded")
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Session{}, err
	}
	id := hex.EncodeToString(raw[:])
	file, err := root.OpenFile(partPath(id), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Session{}, err
	}
	if err := file.Close(); err != nil {
		return Session{}, err
	}
	s := Session{ID: id, Owner: userID, Project: projectID, TaskID: taskID, Name: name, Size: size,
		SHA256: strings.ToLower(hash), Status: "uploading", Expires: time.Now().Add(sessionTTL)}
	if err := save(root, s); err != nil {
		_ = root.Remove(partPath(id))
		return Session{}, err
	}
	return s, nil
}
