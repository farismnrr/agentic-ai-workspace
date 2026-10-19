-- A project has an ID apart from its name, like a Google Cloud project: chosen once, never changed, unique, and
-- the address of its site (/run/<id>/). The name becomes a free label. Projects that already ran a site keep that
-- site's address as their ID; the others get "project-" and the start of their internal id.
ALTER TABLE projects ADD COLUMN slug TEXT NOT NULL DEFAULT '';

UPDATE projects SET slug = (SELECT sites.slug FROM sites WHERE sites.project_id = projects.id)
WHERE EXISTS (SELECT 1 FROM sites WHERE sites.project_id = projects.id);

UPDATE projects SET slug = 'project-' || substr(id, 1, 8) WHERE slug = '';

CREATE UNIQUE INDEX projects_slug_idx ON projects (slug);
