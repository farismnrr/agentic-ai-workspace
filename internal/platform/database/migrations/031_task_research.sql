-- The research notes a team's Researcher writes from the web and the project, shown to the person and shared with the team.
ALTER TABLE tasks ADD COLUMN research TEXT NOT NULL DEFAULT '';
