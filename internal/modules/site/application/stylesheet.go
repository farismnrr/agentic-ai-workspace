package application

import (
	"regexp"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// uncompiled finds what only a build step understands. A stylesheet that still holds it is served as it was
// written, so the browser ignores the rules and the page shows as plain HTML.
var uncompiled = regexp.MustCompile(
	`(?m)@tailwind\s|@import\s+["']tailwindcss|@apply\s|@plugin\s|@theme\s*\{|@source\s`)

// stylesheet checks one stylesheet a page links and follows the files it names with url() and @import.
func (a *auditor) stylesheet(projectID string, ref domain.Reference, inner, pagePath string, got Page) {
	switch {
	case strings.TrimSpace(got.Body) == "":
		a.add(domain.Problem{Link: ref.Raw, OnPage: pagePath, Reason: "is an empty stylesheet, so the page has no styling"})

		return
	case uncompiled.MatchString(got.Body):
		a.add(domain.Problem{
			Link: ref.Raw, OnPage: pagePath,
			Reason: "still holds @tailwind, @apply or @import \"tailwindcss\": it was never built, so the page shows as " +
				"plain HTML; link the stylesheet the build wrote (in dist/, .output/ or .next/), not the source file",
		})

		return
	}

	css := string(domain.Localize(got.Type, "", a.base, a.base, []byte(got.Body)))
	for _, inside := range domain.References(a.base, a.base+inner, css) {
		if path, found := domain.Inside(a.base, inside.Path); found && !domain.IsPage(inside.Path) {
			a.file(projectID, domain.Reference{Raw: inside.Raw + " (in " + ref.Raw + ")", Path: inside.Path}, path, pagePath)
		}
	}
}
