package sandbox

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
)

const (
	maxExtracted = 512 << 20
	sharedDir    = 0o755
	ownerDir     = 0o700
	ownerFile    = 0o600
)

var errArchiveTooBig = errors.New("the archive expands to more than expected")

// extract unpacks a .tar.gz into dir. It goes through os.Root, so an entry (or a symlink
// planted by an earlier entry) can never write outside dir, and it strips setuid bits
// and ownership. Device nodes are skipped; the sandbox gets its devices another way.
func extract(archive, dir string) error {
	file, err := os.Open(archive) //nolint:gosec // the archive is the temporary file we just downloaded
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}

	defer func() { _ = file.Close() }()

	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}

	defer func() { _ = unzipped.Close() }()

	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open %s: %w", dir, err)
	}

	defer func() { _ = root.Close() }()

	reader, total := tar.NewReader(unzipped), int64(0)

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("read archive entry: %w", err)
		}

		if total += header.Size; total > maxExtracted {
			return errArchiveTooBig
		}

		if err := place(root, reader, header); err != nil {
			return fmt.Errorf("unpack %s: %w", header.Name, err)
		}
	}
}

func place(root *os.Root, data io.Reader, header *tar.Header) error {
	name := path.Clean(header.Name)
	if name == "." {
		return nil
	}

	perm := fs.FileMode(header.Mode).Perm() //nolint:gosec // tar modes fit in permission bits

	switch header.Typeflag {
	case tar.TypeDir:
		return wrap(root.MkdirAll(name, perm|ownerDir))
	case tar.TypeSymlink:
		return linkOver(root, name, func() error { return root.Symlink(header.Linkname, name) })
	case tar.TypeLink:
		return linkOver(root, name, func() error { return root.Link(path.Clean(header.Linkname), name) })
	case tar.TypeReg:
		return writeFile(root, data, name, perm|ownerFile)
	}

	return nil
}

func wrap(err error) error {
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	return nil
}

func linkOver(root *os.Root, name string, create func() error) error {
	if err := root.MkdirAll(path.Dir(name), sharedDir); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	_ = root.Remove(name)

	return create()
}

func writeFile(root *os.Root, data io.Reader, name string, perm fs.FileMode) error {
	if err := root.MkdirAll(path.Dir(name), sharedDir); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}

	_ = root.Remove(name)

	file, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(file, data); err != nil { // the total size is capped by the caller
		_ = file.Close()

		return fmt.Errorf("write file: %w", err)
	}

	return file.Close() //nolint:wrapcheck // a close error here is self-explanatory
}
