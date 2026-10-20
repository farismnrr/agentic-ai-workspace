-- The two extra model tiers a person sets once: Thinking (high effort) and Fast. The Default tier stays in the
-- default_provider_id, default_model and default_effort columns. Each is JSON: {"providerId","model","effort"}.
ALTER TABLE preferences ADD COLUMN tier_thinking TEXT NOT NULL DEFAULT '';
ALTER TABLE preferences ADD COLUMN tier_fast TEXT NOT NULL DEFAULT '';
