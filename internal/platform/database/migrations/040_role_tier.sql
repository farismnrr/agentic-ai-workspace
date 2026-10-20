-- Which model tier a role uses when it has no model of its own: thinking, fast, or empty for the default tier.
ALTER TABLE agent_roles ADD COLUMN tier TEXT NOT NULL DEFAULT '';
