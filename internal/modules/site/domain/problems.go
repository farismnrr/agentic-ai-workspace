package domain

import (
	"fmt"
	"strings"
)

// Problems lists what makes the page unusable for the person, as the browser saw it at the app's address: no
// styling, a file the app answered badly, a picture that did not load. Anything from another website is left out,
// because the browser has no internet.
func (c Capture) Problems() []string {
	found := []string{}

	if c.Page.Stylesheets.Rules == 0 {
		found = append(found, "the page has no styling at all: it shows as plain HTML")
	}

	for _, failed := range c.Failed {
		if strings.Contains(failed, " answered ") || strings.HasPrefix(failed, "/") {
			found = append(found, failed)
		}
	}

	if c.Page.Images.Broken > 0 {
		found = append(found, fmt.Sprintf("%d of %d images did not load", c.Page.Images.Broken, c.Page.Images.Total))
	}

	return found
}
