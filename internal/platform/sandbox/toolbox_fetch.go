package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// alpineMirror is where the base filesystem comes from; every request is HTTPS.
const alpineMirror = "https://dl-cdn.alpinelinux.org/alpine"

const (
	fetchTimeout  = 10 * time.Minute
	maxManifest   = 4 << 20
	maxDownload   = 128 << 20
	flavorRootfs  = "alpine-minirootfs"
	errStatusText = "unexpected HTTP status"
)

var (
	errNoRelease = errors.New("no matching Alpine release was listed")
	errChecksum  = errors.New("the download does not match its published checksum")
	errTooBig    = errors.New("the download is larger than expected")
	errStatus    = errors.New(errStatusText)
)

// release is one entry of Alpine's release manifest.
type release struct {
	Version string
	File    string
	SHA256  string
}

func client() *http.Client {
	return &http.Client{Timeout: fetchTimeout, Transport: egress.Transport(egress.Policy{Rule: egress.PublicOnly})}
}

// parseManifest finds the minimal root filesystem for arch in latest-releases.yaml. The
// file is a list of blocks separated by "-" lines, each with simple "key: value" pairs.
func parseManifest(data []byte, arch string) (release, error) {
	for _, block := range strings.Split(string(data), "\n-\n") {
		values := map[string]string{}

		for _, line := range strings.Split(block, "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- ")), ": ")
			if ok {
				values[key] = strings.TrimSpace(value)
			}
		}

		if values["flavor"] == flavorRootfs && values["arch"] == arch && values["file"] != "" && values["sha256"] != "" {
			return release{Version: values["version"], File: values["file"], SHA256: values["sha256"]}, nil
		}
	}

	return release{}, errNoRelease
}

func get(ctx context.Context, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	response, err := client().Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()

		return nil, fmt.Errorf("%w %d from %s", errStatus, response.StatusCode, url)
	}

	return response, nil
}

func fetchRelease(ctx context.Context, mirror, arch string) (release, error) {
	response, err := get(ctx, mirror+"/latest-stable/releases/"+arch+"/latest-releases.yaml")
	if err != nil {
		return release{}, err
	}

	defer func() { _ = response.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxManifest))
	if err != nil {
		return release{}, fmt.Errorf("read manifest: %w", err)
	}

	return parseManifest(data, arch)
}

// download saves the release archive to a temporary file after checking its SHA-256.
func download(ctx context.Context, mirror, arch string, rel release) (string, error) {
	response, err := get(ctx, mirror+"/latest-stable/releases/"+arch+"/"+rel.File)
	if err != nil {
		return "", err
	}

	defer func() { _ = response.Body.Close() }()

	file, err := os.CreateTemp("", "toolbox-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}

	defer func() { _ = file.Close() }()

	sum := sha256.New()

	written, err := io.Copy(io.MultiWriter(file, sum), io.LimitReader(response.Body, maxDownload+1))
	if err != nil {
		_ = os.Remove(file.Name())

		return "", fmt.Errorf("download: %w", err)
	}

	if written > maxDownload {
		_ = os.Remove(file.Name())

		return "", errTooBig
	}

	if hex.EncodeToString(sum.Sum(nil)) != rel.SHA256 {
		_ = os.Remove(file.Name())

		return "", errChecksum
	}

	return file.Name(), nil
}
