package domain

import "time"

// UploadSession is the response to resumable binary transfer tools.
// Opaque upload IDs never grant access without an authenticated owner check.
type UploadSession struct {
	ID      string    `json:"id"`
	Project string    `json:"project"`
	TaskID  string    `json:"task_id"`
	Name    string    `json:"name"`
	Owner   string    `json:"owner,omitempty"`
	Size    int64     `json:"size"`
	Offset  int64     `json:"offset"`
	SHA256  string    `json:"sha256"`
	Status  string    `json:"status"`
	Expires time.Time `json:"expires"`
	Path    string    `json:"path,omitempty"`
}

type FileEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Dir  bool   `json:"dir"`
}

type FileListing struct {
	Entries   []FileEntry `json:"entries"`
	Truncated bool        `json:"truncated"`
}
