package application

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	maxChecked  = 60
	maxPages    = 6
	maxProblems = 8
)

// Page is what the app answered to a request made as the browser would, without a login.
type Page struct {
	Type   string
	Body   string
	Status int
}

// Audit loads the app's home page and a few pages it links to, the way the proxy delivers them, and
// reports links that would not work under the site's address. The proxy moves root links under the address
// itself (domain.Localize), so the audit judges what the person would get: a file that is missing, a web page in
// place of a file, a stylesheet that is empty or was never compiled. The screenshot tool cannot see this: it
// reaches the app without the address prefix, so a link starting with "/" works there and fails for the person.
func (s *Service) Audit(ctx context.Context, site domain.Site) []string {
	base := site.BasePath()
	checker := &auditor{svc: s, ctx: ctx, base: base, seen: map[string]bool{"page /": true}}
	queue := []string{"/"}

	for len(queue) > 0 && len(checker.problems) < maxProblems {
		page := queue[0]
		queue = queue[1:]

		queue = append(queue, checker.scan(site.ProjectID, page)...)
	}

	out := make([]string, 0, len(checker.problems))
	for _, problem := range checker.problems {
		out = append(out, problem.Fix())
	}

	return out
}

type auditor struct {
	svc      *Service
	ctx      context.Context //nolint:containedctx // an auditor lives for one Audit call
	seen     map[string]bool
	base     string
	problems []domain.Problem
	checked  int
	pages    int
}

// scan checks one page's links and returns the pages it links to that are still to be scanned.
func (a *auditor) scan(projectID, pagePath string) []string {
	page, err := a.svc.deps.Runtime.Get(a.ctx, projectID, a.base, pagePath)
	if err != nil || !strings.Contains(page.Type, "html") {
		return nil
	}

	var next []string

	body := string(domain.Localize(page.Type, "", a.base, a.base, []byte(page.Body)))
	for _, ref := range domain.References(a.base, pagePath, body) {
		inner, inside := domain.Inside(a.base, ref.Path)

		ref.Raw = a.written(ref.Raw)

		switch {
		case !inside:
			a.add(domain.Outside(ref.Raw, pagePath))
		case domain.IsPage(ref.Path) && a.pages < maxPages && !a.seen["page "+inner]:
			a.seen["page "+inner] = true
			a.pages++
			next = append(next, inner)
		case !domain.IsPage(ref.Path):
			a.file(projectID, ref, inner, pagePath)
		}
	}

	return next
}

// file requests one file a page needs and reports a missing one, or one answered with a web page.
func (a *auditor) file(projectID string, ref domain.Reference, inner, pagePath string) {
	if a.checked >= maxChecked || a.seen["file "+inner] {
		return
	}

	a.seen["file "+inner] = true
	a.checked++

	got, err := a.svc.deps.Runtime.Get(a.ctx, projectID, a.base, inner)

	switch {
	case err != nil:
	case got.Status >= http.StatusBadRequest:
		a.add(domain.Problem{Link: ref.Raw, OnPage: pagePath, Reason: fmt.Sprintf("answers %d from the app", got.Status)})
	case strings.Contains(got.Type, "html") && !strings.HasSuffix(inner, ".html"):
		a.add(domain.Problem{Link: ref.Raw, OnPage: pagePath, Reason: "answers with a web page, not the file"})
	case strings.Contains(got.Type, "css") || strings.HasSuffix(inner, ".css"):
		a.stylesheet(projectID, ref, inner, pagePath, got)
	}
}

func (a *auditor) add(problem domain.Problem) {
	if len(a.problems) < maxProblems {
		a.problems = append(a.problems, problem)
	}
}

// written gives a link as the app wrote it: the proxy added the address to root links before they were read.
func (a *auditor) written(raw string) string {
	if rest, found := strings.CutPrefix(raw, a.base+"/"); found {
		return "/" + rest
	}

	return raw
}
