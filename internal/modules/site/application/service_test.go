package application_test

import (
	"context"
	"strconv"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

type memStore struct{ sites map[string]domain.Site }

func (m *memStore) Save(_ context.Context, s domain.Site) error { m.sites[s.ProjectID] = s; return nil }

func (m *memStore) ByProject(_ context.Context, id string) (domain.Site, error) {
	if s, ok := m.sites[id]; ok {
		return s, nil
	}

	return domain.Site{}, domain.ErrNotFound
}

func (m *memStore) BySlug(_ context.Context, slug string) (domain.Site, error) {
	for _, s := range m.sites {
		if s.Slug == slug {
			return s, nil
		}
	}

	return domain.Site{}, domain.ErrNotFound
}

func (m *memStore) ByToken(_ context.Context, token string) (domain.Site, error) {
	for _, s := range m.sites {
		if s.Token == token {
			return s, nil
		}
	}

	return domain.Site{}, domain.ErrNotFound
}

func (m *memStore) ListByUser(_ context.Context, user string) ([]domain.Site, error) {
	var out []domain.Site

	for _, s := range m.sites {
		if s.UserID == user {
			out = append(out, s)
		}
	}

	return out, nil
}

func (m *memStore) Running(context.Context) ([]domain.Site, error) {
	var out []domain.Site

	for _, s := range m.sites {
		if s.Status == domain.Running {
			out = append(out, s)
		}
	}

	return out, nil
}

func (m *memStore) SetVisibility(_ context.Context, id string, v domain.Visibility) error {
	s := m.sites[id]
	s.Visibility = v
	m.sites[id] = s

	return nil
}

func (m *memStore) MarkStopped(_ context.Context, id string) error {
	for key, s := range m.sites {
		if id == "" || key == id {
			s.Status, s.ContainerID = domain.Stopped, ""
			m.sites[key] = s
		}
	}

	return nil
}

type fakeRuntime struct {
	failure error
	// output is what the capture command prints.
	output  string
	pages   map[string]application.Page
	stopped []string
	started int
}

func (r *fakeRuntime) Get(_ context.Context, _, _, path string) (application.Page, error) {
	page, ok := r.pages[path]
	if !ok {
		return application.Page{Status: 404, Type: "text/plain"}, nil
	}

	return page, nil
}

func (r *fakeRuntime) Capture(context.Context, application.CaptureRequest) (string, error) {
	return r.output, r.failure
}

func (r *fakeRuntime) Start(context.Context, application.StartSpec) (string, error) {
	if r.failure != nil {
		return "", r.failure
	}

	r.started++

	return "c" + strconv.Itoa(r.started), nil
}
func (r *fakeRuntime) Stop(_ context.Context, id string)  { r.stopped = append(r.stopped, id) }
func (r *fakeRuntime) Alive(context.Context, string) bool { return true }
func (r *fakeRuntime) Socket(string) string               { return "/s" }

type fakeProjects map[string][2]string

func (p fakeProjects) Owner(_ context.Context, id string) (string, string, error) {
	if v, ok := p[id]; ok {
		return v[0], v[1], nil
	}

	return "", "", domain.ErrNotFound
}

type counter struct{ n int }

func (c *counter) Changed(string) { c.n++ }

type seq struct{ n int }

func (s *seq) UUID() (string, error) { s.n++; return "uuid-" + strconv.Itoa(s.n), nil }

func newService(limits application.Limits) (*application.Service, *fakeRuntime, *memStore, *counter) {
	store, runtime, notify := &memStore{sites: map[string]domain.Site{}}, &fakeRuntime{}, &counter{}
	projects := fakeProjects{
		"p1": {"ann", "toko-kopi"}, "p2": {"ann", "blog"}, "p3": {"ann", "third"}, "q1": {"bob", "bobs-shop"},
	}

	return application.NewService(application.Deps{
		Store: store, Runtime: runtime, Projects: projects, Notify: notify, IDs: &seq{},
		Now: time.Now, Limits: limits,
	}), runtime, store, notify
}

var ctx = context.Background()
