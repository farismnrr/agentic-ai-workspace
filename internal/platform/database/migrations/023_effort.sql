-- How hard the model should think: a person's default, and the level each task was started with.
-- An empty value means the provider's own default.
ALTER TABLE preferences ADD COLUMN default_effort TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN effort TEXT NOT NULL DEFAULT '';
