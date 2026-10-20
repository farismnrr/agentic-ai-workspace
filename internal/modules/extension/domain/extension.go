// Package domain holds the rules of browser extensions the AI builds: what a good manifest is, what an extension may
// do in plain words, and what the person is shown before installing it.
package domain

import (
	"errors"
	"time"
)

// Errors of the extension use cases.
var (
	ErrNotFound   = errors.New("no extension was deployed for this project")
	ErrBadFolder  = errors.New("the extension folder must be inside the project")
	ErrNoManifest = errors.New("there is no manifest.json in that folder")
	ErrTooLarge   = errors.New("the extension is too large to package")
)

// Extension is a browser extension the AI built and packaged for the person to install.
type Extension struct {
	DeployedAt time.Time
	ProjectID  string
	UserID     string
	// Slug is the project's ID, used to name the package file.
	Slug        string
	Name        string
	Version     string
	Description string
	// Permissions and Hosts are what the manifest asks for, as written there.
	Permissions []string
	Hosts       []string
	Size        int64
	Files       int
}

// FileName is the name the person's download gets.
func (e Extension) FileName() string { return e.Slug + "-" + e.Version + ".zip" }
