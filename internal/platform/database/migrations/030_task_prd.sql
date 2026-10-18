-- The product requirements document a team's Planner writes, shown to the person and shared with the team.
ALTER TABLE tasks ADD COLUMN prd TEXT NOT NULL DEFAULT '';
