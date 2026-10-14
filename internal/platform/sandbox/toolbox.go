package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// markerFile proves a toolbox was installed completely: it is written last.
const markerFile = ".toolbox.json"

// mountPoints must exist in the toolbox, because nothing can be created on its read-only root.
var mountPoints = []string{"proc", "dev", "tmp", "workspace"}

// DefaultPackages are installed into the toolbox when the owner does not choose others.
// Names are Alpine Linux packages.
const DefaultPackages = "git nodejs npm python3 go rust cargo make build-base"

var (
	packageName = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]{0,63}$`)

	errBadPackage = errors.New("not a valid package name")
)

// Marker describes an installed toolbox.
type Marker struct {
	InstalledAt time.Time `json:"installedAt"`
	Alpine      string    `json:"alpine"`
	Packages    []string  `json:"packages"`
}

// InstallOptions say what to build and where.
type InstallOptions struct {
	// Out receives progress and the package manager's output.
	Out io.Writer
	// Dir is the toolbox folder; an existing one is replaced only after the new one is complete.
	Dir string
	// Mirror is the Alpine mirror; empty means the official one.
	Mirror   string
	Packages []string
}

// ToolboxReady reports whether dir holds a complete toolbox.
func ToolboxReady(dir string) error {
	if _, err := ReadMarker(dir); err != nil {
		return err
	}

	for _, name := range mountPoints {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.IsDir() {
			return fmt.Errorf("%w: %s is missing, so install it again", ErrNoToolbox, name)
		}
	}

	return nil
}

// ReadMarker returns what the toolbox at dir was built from.
func ReadMarker(dir string) (Marker, error) {
	//nolint:gosec // dir is the configured toolbox folder
	data, err := os.ReadFile(filepath.Join(dir, markerFile))
	if err != nil {
		return Marker{}, ErrNoToolbox
	}

	var marker Marker
	if err := json.Unmarshal(data, &marker); err != nil {
		return Marker{}, fmt.Errorf("%w: %w", ErrNoToolbox, err)
	}

	return marker, nil
}

// ParsePackages splits a space-separated package list and rejects anything that is
// not a plain package name, since the names become arguments of the installer.
func ParsePackages(list string) ([]string, error) {
	names := strings.Fields(list)

	for _, name := range names {
		if !packageName.MatchString(name) {
			return nil, fmt.Errorf("%w: %q", errBadPackage, name)
		}
	}

	return names, nil
}
