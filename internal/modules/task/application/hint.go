package application

import (
	"encoding/json"
	"strings"
)

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

// commandHold is why a call may not run whatever the person says: it can only end in a server nobody can reach or a
// font nobody can load.
func (r *Runner) commandHold(call Call) string {
	if why := r.serverHold(call); why != "" {
		return why
	}

	return r.fontLinkHold(call)
}

// fontLinkHold says why a file may not be written: it links a font from a website, which no build or screenshot in
// the sandbox can load. The person's browser could, but the font would then be unpinned and unseen by the team.
func (r *Runner) fontLinkHold(call Call) string {
	var in struct {
		Content string `json:"content"`
		NewText string `json:"new_text"`
	}

	if call.Name != "write_file" && call.Name != "edit_file" || json.Unmarshal(call.Input, &in) != nil {
		return ""
	}

	text := strings.ToLower(in.Content + "\n" + in.NewText)
	for _, host := range fontHosts {
		if strings.Contains(text, host) {
			return "This file links a font from " + host + "." + fontHint
		}
	}

	return ""
}
