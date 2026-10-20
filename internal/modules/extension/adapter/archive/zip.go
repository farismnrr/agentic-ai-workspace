package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"path"
)

func writeZip(out io.Writer, source fs.FS) error {
	archive := zip.NewWriter(out)

	err := fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !entry.Type().IsRegular() || skipped(name) {
			return walkErr
		}

		return addFile(archive, source, name)
	})
	if err != nil {
		return fmt.Errorf("zip the folder: %w", err)
	}

	if err = archive.Close(); err != nil {
		return fmt.Errorf("finish the zip: %w", err)
	}

	return nil
}

func addFile(archive *zip.Writer, source fs.FS, name string) error {
	file, err := source.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("describe %s: %w", name, err)
	}

	header.Name, header.Method = name, zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}

	if _, err = io.Copy(writer, file); err != nil {
		return fmt.Errorf("copy %s: %w", name, err)
	}

	return nil
}

// skipped are files an operating system leaves behind, which do not belong in a package.
func skipped(name string) bool {
	base := path.Base(name)

	return base == ".DS_Store" || base == "Thumbs.db"
}
