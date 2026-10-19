-- A site's address is now /run/<slug>/ and the slug is unique, so the project's name is what the person sees
-- in the address. Addresses made before this carried a UUID after the slug (the token); the token stays in the
-- table and the old addresses are redirected. Sites that shared a slug keep it on the oldest row only; the others
-- get the start of their token added so the unique index can be made.
UPDATE sites SET slug = slug || '-' || substr(token, 1, 6)
WHERE rowid NOT IN (SELECT MIN(rowid) FROM sites GROUP BY slug);

CREATE UNIQUE INDEX sites_slug_idx ON sites (slug);
