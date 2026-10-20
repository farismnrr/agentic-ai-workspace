-- Whether a person has switched one of their own plans (Claude, ChatGPT) off. A plan with no row is on, so
-- signing in needs no row; only the switch writes one.
CREATE TABLE plan_switches (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    program TEXT NOT NULL,
    enabled INTEGER NOT NULL,
    PRIMARY KEY (user_id, program)
);
