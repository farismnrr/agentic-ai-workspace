package files

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"
	"time"
)

func projectUsage(root *os.Root) (int64, error) {
	dir, err := root.Open("uploads")
	if err != nil {
		return 0, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return 0, err
	}
	var size int64
	for _, item := range entries {
		if item.IsDir() {
			continue
		}
		info, err := item.Info()
		if err != nil {
			return 0, err
		}
		size += info.Size()
	}
	// Reserve capacity for live sessions, not only chunks already uploaded.
	staged, err := root.Open(stage)
	if err != nil {
		return 0, err
	}
	defer staged.Close()
	items, err := staged.ReadDir(-1)
	if err != nil {
		return 0, err
	}
	active := 0
	for _, item := range items {
		if !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		data, err := root.ReadFile(path.Join(stage, item.Name()))
		if err != nil {
			return 0, err
		}
		var session Session
		if json.Unmarshal(data, &session) != nil {
			continue
		}
		if session.Status != "uploading" {
			continue
		}
		if time.Now().After(session.Expires) {
			_ = root.Remove(partPath(session.ID))
			_ = root.Remove(metaPath(session.ID))
			continue
		}
		active++
		size += session.Size
	}
	if active >= 8 {
		return 0, errors.New("too many active uploads for this project")
	}
	return size, nil
}
