package fs

import (
	"syscall"
)

// AvailableUploadBytes reports free bytes on the confined project's filesystem,
// not the host root filesystem. The project upload quota is checked separately.
func (r *Root) AvailableUploadBytes() (int64, error) {
	file, err := r.root.Open(".")
	if err != nil {
		return 0, translate(err)
	}
	defer func() { _ = file.Close() }()
	var stat syscall.Statfs_t
	if err := syscall.Fstatfs(int(file.Fd()), &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
