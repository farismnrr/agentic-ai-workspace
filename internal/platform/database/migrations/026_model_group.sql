-- Who a gateway says serves each model (9router reports it as owned_by: cc for Claude Code, cx for Codex, kr for
-- Kiro, the name of a combo, ...), so models can be shown in groups. Empty when the provider said nothing.
ALTER TABLE model_catalog ADD COLUMN owner TEXT NOT NULL DEFAULT '';
