package domain

import (
	"errors"
	"regexp"
	"strings"
)

// Screenshot errors.
var (
	ErrNotRunning = errors.New("no app is running for this project; start it first")
	ErrBadPath    = errors.New("use a page address that starts with /, such as /pricing")
	ErrNoPicture  = errors.New("the browser could not take the picture")
)

// Window sizes of the two kinds of device, in CSS pixels.
const (
	DesktopWidth  = 1280
	DesktopHeight = 800
	MobileWidth   = 390
	MobileHeight  = 844
	maxPathLength = 200
	maxOffset     = 100_000
)

var pagePath = regexp.MustCompile(`^/[A-Za-z0-9/_.~%+=&?#-]*$`)

// CaptureSpec says which page of the running app to photograph and how.
type CaptureSpec struct {
	// Path is the page, such as /pricing; the app is always the project's own.
	Path string
	// Y is how far down the page the picture starts, for a closer look at a long page.
	Y int
	// Overview shows the whole page scaled down instead of one screen.
	Overview bool
	Mobile   bool
}

// Normalize checks the path and returns it with a default.
func (c CaptureSpec) Normalize() (CaptureSpec, error) {
	if c.Path == "" {
		c.Path = "/"
	}

	if len(c.Path) > maxPathLength || !pagePath.MatchString(c.Path) || strings.Contains(c.Path, "..") ||
		strings.HasPrefix(c.Path, "//") {
		return CaptureSpec{}, ErrBadPath
	}

	c.Y = min(max(c.Y, 0), maxOffset)

	return c, nil
}

// Window is the size of the screen the page is laid out for.
func (c CaptureSpec) Window() (width, height int) {
	if c.Mobile {
		return MobileWidth, MobileHeight
	}

	return DesktopWidth, DesktopHeight
}

// Frame is the part of the page the picture shows.
type Frame struct {
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float64 `json:"scale"`
}

// Page is what the page says about itself, read in the browser.
type Page struct {
	Title        string   `json:"title"`
	Lang         string   `json:"lang"`
	Font         string   `json:"font"`
	Text         string   `json:"text"`
	Headings     []string `json:"headings"`
	Landmarks    []string `json:"landmarks"`
	LowContrast  []string `json:"lowContrast"`
	Width        int      `json:"width"`
	PageWidth    int      `json:"pageWidth"`
	PageHeight   int      `json:"pageHeight"`
	SmallTargets int      `json:"smallTargets"`
	Unlabeled    int      `json:"unlabeledFields"`
	Images       struct {
		Total  int `json:"total"`
		Broken int `json:"broken"`
		NoAlt  int `json:"noAlt"`
	} `json:"images"`
	Stylesheets struct {
		Total int `json:"total"`
		Rules int `json:"rules"`
	} `json:"stylesheets"`
	ViewportMeta bool `json:"viewportMeta"`
	OverflowX    bool `json:"overflowX"`
}

// Capture is a picture of a page and what was found on it.
type Capture struct {
	PNG     []byte
	Console []string
	Failed  []string
	Page    Page
	Frame   Frame
	Status  int
}
