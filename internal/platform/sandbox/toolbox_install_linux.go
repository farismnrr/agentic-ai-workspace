package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	sharedFile  = 0o644
	ownerAccess = 0o700
)

var alpineArch = map[string]string{"amd64": "x86_64", "arm64": "aarch64"}

// Install builds the toolbox: Alpine Linux's minimal root filesystem (checksum verified)
// plus the packages, installed by Alpine's own package manager inside an unprivileged
// namespace. The result is its own world: nothing in it comes from the host.
func Install(ctx context.Context, opts InstallOptions) error {
	arch, ok := alpineArch[runtime.GOARCH]
	if !ok {
		return ErrUnsupported
	}

	mirror := opts.Mirror
	if mirror == "" {
		mirror = alpineMirror
	}

	stage := opts.Dir + ".new"
	if err := resetDir(stage); err != nil {
		return err
	}

	defer func() { _ = removeAll(stage) }()

	say(opts.Out, "Looking up the latest Alpine Linux release…")

	rel, err := fetchRelease(ctx, mirror, arch)
	if err != nil {
		return err
	}

	say(opts.Out, "Downloading Alpine %s and checking its checksum…", rel.Version)

	archive, err := download(ctx, mirror, arch, rel)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(archive) }()

	if err = extract(archive, stage); err != nil {
		return err
	}

	if err = installPackages(ctx, stage, opts); err != nil {
		return err
	}

	marker := Marker{InstalledAt: time.Now().UTC(), Alpine: rel.Version, Packages: opts.Packages}
	if err = finish(stage, marker); err != nil {
		return err
	}

	return swap(stage, opts.Dir)
}

func installPackages(ctx context.Context, stage string, opts InstallOptions) error {
	for _, name := range mountPoints {
		if err := os.MkdirAll(filepath.Join(stage, name), dirMode); err != nil {
			return fmt.Errorf("create /%s: %w", name, err)
		}
	}

	if len(opts.Packages) == 0 {
		return nil
	}

	if err := copyResolver(stage); err != nil {
		return err
	}

	say(opts.Out, "Installing %d packages (this can take several minutes)…", len(opts.Packages))

	argv := append([]string{"/sbin/apk", "add", "--no-cache"}, opts.Packages...)
	if err := buildCommand(ctx, stage, argv, opts.Out).Run(); err != nil {
		return fmt.Errorf("install packages: %w", err)
	}

	return nil
}

// copyResolver lets the package manager resolve names; it is removed before the toolbox is finished.
func copyResolver(stage string) error {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return fmt.Errorf("read host resolver settings: %w", err)
	}

	//nolint:gosec // the toolbox is world-readable by design
	if err := os.WriteFile(filepath.Join(stage, "etc", "resolv.conf"), data, sharedFile); err != nil {
		return fmt.Errorf("write resolver settings: %w", err)
	}

	return nil
}

// finish removes install-time leftovers and writes the marker last, so a half-built toolbox is never used.
func finish(stage string, marker Marker) error {
	for _, leftover := range []string{"etc/resolv.conf", "dev/null", "dev/zero", "dev/random", "dev/urandom"} {
		_ = os.Remove(filepath.Join(stage, leftover))
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("encode marker: %w", err)
	}

	if err := os.WriteFile(filepath.Join(stage, markerFile), data, sharedFile); err != nil {
		return fmt.Errorf("write marker: %w", err)
	}

	return nil
}

// swap puts the finished toolbox in place. A toolbox that sandboxes are still using keeps working:
// they hold the old directory open until they end.
func swap(stage, dir string) error {
	old := dir + ".old"
	if err := removeAll(old); err != nil {
		return err
	}

	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, old); err != nil {
			return fmt.Errorf("move the old toolbox aside: %w", err)
		}
	}

	if err := os.Rename(stage, dir); err != nil {
		return fmt.Errorf("put the toolbox in place: %w", err)
	}

	return removeAll(old)
}

func resetDir(dir string) error {
	if err := removeAll(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	return nil
}

// removeAll deletes a tree even when package files made some folders unwritable.
func removeAll(dir string) error {
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			_ = os.Chmod(path, ownerAccess) //nolint:gosec // directories we created, being deleted
		}

		return nil
	})

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove %s: %w", dir, err)
	}

	return nil
}

func say(out io.Writer, format string, args ...any) {
	if out != nil {
		_, _ = fmt.Fprintf(out, format+"\n", args...)
	}
}
