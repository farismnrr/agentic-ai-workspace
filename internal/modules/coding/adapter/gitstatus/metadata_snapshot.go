package gitstatus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type metadataSnapshot struct {
	gitDir       string
	worktree     string
	objectFormat string
}

const (
	privateDirectoryMode os.FileMode = 0o700
	privateFileMode      os.FileMode = 0o600
)

func createMetadataSnapshot(worktree string) (*metadataSnapshot, error) {
	absWorktree, err := filepath.Abs(worktree)
	if err != nil {
		return nil, fmt.Errorf("resolve project worktree: %w", err)
	}
	project, err := os.OpenRoot(absWorktree)
	if err != nil {
		return nil, fmt.Errorf("open project worktree: %w", err)
	}
	defer func() { _ = project.Close() }()
	gitInfo, err := project.Lstat(".git")
	if err != nil {
		return nil, fmt.Errorf("inspect project Git metadata: %w", err)
	}
	if !gitInfo.IsDir() || gitInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errGitMetadata
	}
	gitRoot, err := project.OpenRoot(".git")
	if err != nil {
		return nil, fmt.Errorf("open project Git metadata: %w", err)
	}
	defer func() { _ = gitRoot.Close() }()
	openedInfo, err := gitRoot.Stat(".")
	if err != nil || !sameMetadata(gitInfo, openedInfo) {
		return nil, errGitMetadataRace
	}
	return prepareMetadataSnapshot(project, gitRoot, gitInfo, absWorktree)
}

func prepareMetadataSnapshot(
	project, gitRoot *os.Root, gitInfo os.FileInfo, worktree string,
) (*metadataSnapshot, error) {
	directory, createErr := os.MkdirTemp("", "gitstatus-snapshot-")
	if createErr != nil {
		return nil, fmt.Errorf("create private Git snapshot: %w", createErr)
	}
	snapshot := &metadataSnapshot{gitDir: directory, worktree: worktree}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(directory)
		}
	}()
	copier := newMetadataCopier(gitRoot, directory)
	safeConfig, configErr := prepareSnapshotConfig(snapshot, copier)
	if configErr != nil {
		return nil, configErr
	}
	if copyErr := copyMetadataEntries(copier, safeConfig); copyErr != nil {
		return nil, copyErr
	}
	snapshot.objectFormat = safeConfig.objectFormat
	if writeErr := writeSafeGitConfig(snapshot, copier, safeConfig); writeErr != nil {
		return nil, writeErr
	}
	if verifyErr := copier.verify(); verifyErr != nil {
		return nil, verifyErr
	}
	currentInfo, inspectErr := project.Lstat(".git")
	if inspectErr != nil || !sameMetadata(gitInfo, currentInfo) {
		return nil, errGitMetadataRace
	}
	keep = true
	return snapshot, nil
}

func prepareSnapshotConfig(snapshot *metadataSnapshot, copier *metadataCopier) (safeGitConfig, error) {
	if _, err := copyRawConfig(copier); err != nil {
		return safeGitConfig{}, err
	}
	if err := prepareConfigRepository(snapshot.gitDir); err != nil {
		return safeGitConfig{}, err
	}
	return preflightGitConfig(snapshot)
}

func prepareConfigRepository(gitDir string) error {
	objectDir := filepath.Join(gitDir, "objects")
	if err := os.Mkdir(objectDir, privateDirectoryMode); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("prepare Git config snapshot: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), privateDirectoryMode); err != nil {
		return fmt.Errorf("prepare Git config refs: %w", err)
	}
	head := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), head, privateFileMode); err != nil {
		return fmt.Errorf("prepare Git config HEAD: %w", err)
	}
	return nil
}

func copyRawConfig(copier *metadataCopier) ([]byte, error) {
	data, configErr := copier.copyFile("config", maxGitConfigBytes)
	if errors.Is(configErr, os.ErrNotExist) {
		if countErr := copier.countEntry(); countErr != nil {
			return nil, countErr
		}
		if writeErr := os.WriteFile(filepath.Join(copier.target, "config"), nil, privateFileMode); writeErr != nil {
			return nil, fmt.Errorf("create snapshot Git config: %w", writeErr)
		}
		return nil, nil
	}
	if configErr != nil {
		return nil, errGitConfigUnsupported
	}
	return data, nil
}

func writeSafeGitConfig(snapshot *metadataSnapshot, copier *metadataCopier, config safeGitConfig) error {
	contents := config.contents()
	if err := copier.addBytes(int64(len(contents))); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(snapshot.gitDir, "config"), contents, privateFileMode); err != nil {
		return fmt.Errorf("write safe Git snapshot config: %w", err)
	}
	return nil
}

func (snapshot *metadataSnapshot) Close() error {
	if err := os.RemoveAll(snapshot.gitDir); err != nil {
		return fmt.Errorf("remove private Git snapshot: %w", err)
	}
	return nil
}
