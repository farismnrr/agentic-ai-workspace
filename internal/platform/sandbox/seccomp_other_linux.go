//go:build linux && !amd64 && !arm64

package sandbox

// installFilter fails closed: without a filter for this CPU, tasks must not run.
func installFilter() error {
	return ErrUnsupported
}
