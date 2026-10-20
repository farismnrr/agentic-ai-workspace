-- The context window a person chose for one model of their own plans (Claude, ChatGPT). A model with no row
-- keeps the size its program reports.
CREATE TABLE plan_windows (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    program TEXT NOT NULL,
    model TEXT NOT NULL,
    context_window INTEGER NOT NULL,
    PRIMARY KEY (user_id, program, model)
);
