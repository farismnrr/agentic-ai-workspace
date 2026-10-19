package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/moby/api/types/mount"
)

var errOutsideData = errors.New("workspace is outside the data folder")

// folderMount shows one folder of the data directory at target. Run directly on the
// host the folder is bind-mounted; run inside a container, the daemon cannot see
// the server's paths, so the data volume is mounted and narrowed to the sub-path.
func (c Config) folderMount(dir, target string, readOnly bool) (mount.Mount, error) {
	if c.DataVolume == "" {
		return mount.Mount{Type: mount.TypeBind, Source: dir, Target: target, ReadOnly: readOnly}, nil
	}

	rel, err := filepath.Rel(c.DataDir, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return mount.Mount{}, fmt.Errorf("%w: %s", errOutsideData, dir)
	}

	return mount.Mount{
		Type: mount.TypeVolume, Source: c.DataVolume, Target: target, ReadOnly: readOnly,
		VolumeOptions: &mount.VolumeOptions{Subpath: filepath.ToSlash(rel)},
	}, nil
}

// mounts lists what a command may see besides its image: the workspace, and the cache and proxy when given.
func (c Config) mounts(spec Spec) ([]mount.Mount, error) {
	folders := []struct {
		dir, target string
		readOnly    bool
	}{
		{spec.Workspace, Workspace, false},
		{spec.Cache, Cache, false},
		{spec.Egress, Egress, true},
		{spec.Site, SiteDir, false},
		{spec.Home, Home, false},
	}

	var out []mount.Mount

	for _, folder := range folders {
		if folder.dir == "" {
			continue
		}

		m, err := c.folderMount(folder.dir, folder.target, folder.readOnly)
		if err != nil {
			return nil, err
		}

		out = append(out, m)
	}

	return out, nil
}
