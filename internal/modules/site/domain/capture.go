package domain

import (
	"errors"
	"fmt"
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
	ViewportMeta bool `json:"viewportMeta"`
	OverflowX    bool `json:"overflowX"`
}

// Capture is a picture of a page and what was found on it.
type Capture struct {
	Console []string
	Failed  []string
	PNG     []byte
	Page    Page
	Frame   Frame
}

// Describe tells the model what the picture does not: the facts of the page, and what to look at.
func (c Capture) Describe(spec CaptureSpec) string {
	var out strings.Builder

	device := "desktop"
	if spec.Mobile {
		device = "phone"
	}

	p := c.Page
	fmt.Fprintf(&out, "Screenshot of %s on a %s screen (%dpx wide). ", spec.Path, device, c.Frame.Width)

	if spec.Overview {
		fmt.Fprintf(&out, "It shows the whole page, %dpx tall, scaled to %.0f%%.\n", p.PageHeight, c.Frame.Scale*percent)
	} else {
		fmt.Fprintf(&out, "It shows %dpx from %dpx down a page that is %dpx tall; ask with y for another part.\n",
			c.Frame.Height, c.Frame.Y, p.PageHeight)
	}

	fmt.Fprintf(&out, "Title: %q. Language: %q. Font: %s.\n", p.Title, p.Lang, p.Font)
	c.writeProblems(&out)
	list(&out, "Headings", p.Headings)
	list(&out, "Landmarks", p.Landmarks)
	fmt.Fprintf(&out, "Text: %s\n", p.Text)
	out.WriteString("The browser has no internet: fonts or images loaded from other sites show as failed here but " +
		"will load for the person.")

	return out.String()
}

const percent = 100

func (c Capture) writeProblems(out *strings.Builder) {
	p := c.Page
	found := []string{}

	if p.OverflowX {
		found = append(found, fmt.Sprintf("the page is %dpx wide on a %dpx screen, so it scrolls sideways",
			p.PageWidth, p.Width))
	}

	if !p.ViewportMeta {
		found = append(found, "no <meta name=viewport>, so phones show it zoomed out")
	}

	if p.Images.Broken > 0 {
		found = append(found, fmt.Sprintf("%d of %d images did not load", p.Images.Broken, p.Images.Total))
	}

	if p.Images.NoAlt > 0 {
		found = append(found, fmt.Sprintf("%d images have no alt text", p.Images.NoAlt))
	}

	if p.SmallTargets > 0 {
		found = append(found, fmt.Sprintf("%d buttons or links are smaller than 24px", p.SmallTargets))
	}

	if p.Unlabeled > 0 {
		found = append(found, fmt.Sprintf("%d form fields have no label", p.Unlabeled))
	}

	if p.Lang == "" {
		found = append(found, "<html> has no lang")
	}

	if len(found) == 0 {
		out.WriteString("Checks: nothing found by the automatic checks (they do not replace looking).\n")
	} else {
		out.WriteString("Checks found: " + strings.Join(found, "; ") + ".\n")
	}

	list(out, "Low contrast text", p.LowContrast)
	list(out, "Console", c.Console)
	list(out, "Requests that failed", c.Failed)
}

func list(out *strings.Builder, name string, items []string) {
	if len(items) > 0 {
		out.WriteString(name + ": " + strings.Join(items, " | ") + "\n")
	}
}
