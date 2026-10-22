package domain

import (
	"fmt"
	"strings"
)

const percent = 100

// Describe tells the model what the picture does not: the facts of the page, and what to look at.
func (c Capture) Describe(spec CaptureSpec) string {
	var out strings.Builder
	device := "desktop"
	if spec.Mobile {
		device = "phone"
	}
	p := c.Page
	if c.Status != 0 {
		fmt.Fprintf(&out, "Page response: HTTP %d.\n", c.Status)
	}
	fmt.Fprintf(&out, "Screenshot of %s on a %s screen (%dpx wide). ", safeCapturePath(spec.Path), device, c.Frame.Width)
	if spec.Overview {
		fmt.Fprintf(&out, "It shows the whole page, %dpx tall, scaled to %.0f%%.\n",
			p.PageHeight, c.Frame.Scale*percent)
	} else {
		fmt.Fprintf(&out, "It shows %dpx from %dpx down a page that is %dpx tall; ask with y for another part.\n",
			c.Frame.Height, c.Frame.Y, p.PageHeight)
	}
	fmt.Fprintf(&out, "Title: %q. Language: %q. Font: %s.\n", safeCaptureText(p.Title), p.Lang, safeCaptureText(p.Font))
	c.writeProblems(&out)
	list(&out, "Headings", p.Headings)
	list(&out, "Landmarks", p.Landmarks)
	fmt.Fprintf(&out, "Text: %s\n", safeCaptureText(p.Text))
	out.WriteString("The browser has no internet: fonts or images loaded from other sites show as failed here but " +
		"will load for the person.")
	return out.String()
}

func (c Capture) writeProblems(out *strings.Builder) {
	p := c.Page
	found := []string{}
	if p.OverflowX {
		found = append(found, fmt.Sprintf(
			"the page is %dpx wide on a %dpx screen, so it scrolls sideways", p.PageWidth, p.Width,
		))
	}
	if !p.ViewportMeta {
		found = append(found, "no <meta name=viewport>, so phones show it zoomed out")
	}
	if p.Stylesheets.Rules == 0 {
		found = append(found, "NO STYLESHEET APPLIED: the page shows as plain HTML. Its stylesheet is missing, empty, "+
			"never built or answered with a web page (see the failed requests); fix this before anything else")
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
	if len(items) == 0 {
		return
	}
	safe := make([]string, 0, len(items))
	for _, item := range items {
		safe = append(safe, safeCaptureText(item))
	}
	out.WriteString(name + ": " + strings.Join(safe, " | ") + "\n")
}
