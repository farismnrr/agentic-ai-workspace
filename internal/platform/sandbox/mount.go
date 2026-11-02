package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/api/types/mount"
)

// ErrOutsideData means a volume-backed sandbox path escapes the configured data directory.
var ErrOutsideData = errors.New("workspace is outside the data folder")

// folderMount shows one folder of the data directory at target. Run directly on the
// host the folder is bind-mounted; run inside a container, the daemon cannot see
// the server's paths, so the data volume is mounted and narrowed to the sub-path.
func (c Config) folderMount(dir, target string, readOnly bool) (mount.Mount, error) {
	var err error
	if dir, err = canonicalMountPath(dir); err != nil {
		return mount.Mount{}, err
	}
	if c.DataVolume == "" {
		return mount.Mount{Type: mount.TypeBind, Source: dir, Target: target, ReadOnly: readOnly}, nil
	}
	dataDir, err := canonicalMountPath(c.DataDir)
	if err != nil {
		return mount.Mount{}, err
	}

	rel, err := filepath.Rel(dataDir, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return mount.Mount{}, fmt.Errorf("%w: %s", ErrOutsideData, dir)
	}

	return mount.Mount{
		Type: mount.TypeVolume, Source: c.DataVolume, Target: target, ReadOnly: readOnly,
		VolumeOptions: &mount.VolumeOptions{Subpath: filepath.ToSlash(rel)},
	}, nil
}

// mounts lists what a command may see besides its image: the workspace, and the cache and proxy when given.
// Mounts validates source paths and builds the requested container mounts.
func (c Config) Mounts(spec Spec) ([]mount.Mount, error) {
	folders := []struct {
		dir, target string
		readOnly    bool
	}{
		{spec.Workspace, Workspace, spec.WorkspaceReadOnly},
		{spec.Cache, Cache, false},
		{spec.Egress, Egress, true},
		{spec.Site, SiteDir, false},
		{spec.View, ViewDir, false},
		{spec.Home, Home, false},
		{spec.Bridge, BridgeDir, false},
		{spec.Artifacts, BuildOutput, false},
	}

	var out []mount.Mount

	for _, folder := range folders {
		if folder.dir == "" {
			continue
		}

		m, err := c.folderMount(folder.dir, folder.target, folder.readOnly)
		if err != nil {
			return nil, errors.New("sandbox mount source is unavailable or unsafe")
		}

		out = append(out, m)
	}

	if len(spec.WorkspaceWritable) > 0 && !spec.WorkspaceReadOnly {
		return nil, fmt.Errorf("Sites writable paths require a read-only workspace")
	}
	if len(spec.WorkspaceWritable) > 0 && !withinDataDir(c.DataDir, spec.Workspace) {
		return nil, fmt.Errorf("Sites workspace storage is unavailable; ask the administrator to check storage")
	}

	for i, writable := range spec.WorkspaceWritable {
		if err := validateWorkspaceMount(spec.WorkspaceWritable[:i], writable); err != nil {
			return nil, err
		}
		rel := filepath.FromSlash(writable)
		target := filepath.Join(Workspace, rel)
		source := filepath.Join(spec.Workspace, rel)
		m, err := c.folderMount(source, target, false)
		if err != nil {
			return nil, fmt.Errorf("Sites storage unavailable for workspace path %q; ask the administrator to check storage", writable)
		}
		out = append(out, m)
	}

	return out, nil
}

func withinDataDir(dataDir, target string) bool {
	dataDir, dataErr := canonicalMountPath(dataDir)
	target, targetErr := canonicalMountPath(target)
	if dataErr != nil || targetErr != nil {
		return false
	}
	rel, err := filepath.Rel(dataDir, target)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func canonicalMountPath(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", errors.New("mount path must be absolute and clean")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return "", errors.New("mount path contains a symlink or is unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("mount path is not a directory")
	}
	return resolved, nil
}

func validateWorkspaceMount(previous []string, current string) error {
	rel := filepath.FromSlash(current)
	if current == "" || strings.Contains(current, `\`) || filepath.IsAbs(rel) || filepath.Clean(rel) != rel ||
		rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid Sites writable path %q", current)
	}

	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("invalid Sites writable path %q", current)
		}
	}
	if current != "data/preview-runs" && current != ".atja-runtime" {
		return fmt.Errorf("Sites writable path %q is not approved", current)
	}

	for _, prior := range previous {
		if current == prior || strings.HasPrefix(current, prior+"/") ||
			strings.HasPrefix(prior, current+"/") {
			return fmt.Errorf("overlapping Sites writable paths %q and %q", prior, current)
		}
	}

	return nil
}
