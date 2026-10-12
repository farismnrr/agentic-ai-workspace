package webassets

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed dist/*
var embeddedAssets embed.FS

// Assets returns the compiled SPA as an embedded filesystem.
func Assets() (fs.FS, error) {
	assets, err := fs.Sub(embeddedAssets, "dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded SPA: %w", err)
	}

	return assets, nil
}
