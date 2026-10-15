-- How a gateway says a model thinks (9router: openai, claude-adaptive, claude-budget, gemini-level, ...). The effort
-- levels worth offering depend on it: a Claude-style model has no "minimal" or "none". Empty when not reported.
ALTER TABLE model_catalog ADD COLUMN format TEXT NOT NULL DEFAULT '';
