-- The command a site was last started with, so the person can start it again from the Sites page without the AI.
-- The address (token) is now kept for the life of the project instead of being replaced on every start.
ALTER TABLE sites ADD COLUMN command TEXT NOT NULL DEFAULT '';
