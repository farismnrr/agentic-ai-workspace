package application

import "strings"

// fontHosts are the font websites a command cannot reach: only package registries are open to it.
var fontHosts = []string{"fonts.googleapis.com", "fonts.gstatic.com", "use.typekit.net", "fonts.bunny.net"}

const fontHint = "\n\nFonts cannot be downloaded from a website here; only package registries are reachable. Install " +
	"the font from npm instead: the Fontsource package @fontsource-variable/<name> (or @fontsource/<name>), imported " +
	"in the stylesheet, or its woff2 files copied into the project with an @font-face."

// withHint is what a tool printed, with a way forward added when a failure shows a known dead end.
func withHint(result ToolResult) string {
	if !result.IsError {
		return result.Output
	}

	text := strings.ToLower(result.Output)
	for _, host := range fontHosts {
		if strings.Contains(text, host) {
			return result.Output + fontHint
		}
	}

	return result.Output
}
