package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moby/moby/api/types/mount"
)

var errOutsideData = errors.New("workspace is outside the data folder")

// workspaceMount shows one workspace folder at /workspace. Run directly on the
// host the folder is bind-mounted; run inside a container, the daemon cannot see
// the server's paths, so the data volume is mounted and narrowed to the sub-path.
func (c Config) workspaceMount(dir string) (mount.Mount, error) {
	if c.DataVolume == "" {
		return mount.Mount{Type: mount.TypeBind, Source: dir, Target: Workspace}, nil
	}

	rel, err := filepath.Rel(c.DataDir, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return mount.Mount{}, fmt.Errorf("%w: %s", errOutsideData, dir)
	}

	return mount.Mount{
		Type: mount.TypeVolume, Source: c.DataVolume, Target: Workspace,
		VolumeOptions: &mount.VolumeOptions{Subpath: filepath.ToSlash(rel)},
	}, nil
}
