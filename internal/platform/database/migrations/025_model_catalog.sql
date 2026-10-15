-- What a provider told us about each of its models when the key was verified: the context window, and how the
-- model thinks (whether it reasons at all, whether it can switch thinking off, and the effort levels it lists
-- itself). It replaces model_windows, which only held the window. A zero window or reported = 0 means the
-- provider said nothing.
CREATE TABLE model_catalog (
    provider_id    TEXT    NOT NULL REFERENCES providers (id) ON DELETE CASCADE,
    model          TEXT    NOT NULL,
    context_window INTEGER NOT NULL DEFAULT 0,
    reported       INTEGER NOT NULL DEFAULT 0,
    reasoning      INTEGER NOT NULL DEFAULT 0,
    can_disable    INTEGER NOT NULL DEFAULT 1,
    levels         TEXT    NOT NULL DEFAULT '[]',
    PRIMARY KEY (provider_id, model)
);

INSERT INTO model_catalog (provider_id, model, context_window)
SELECT provider_id, model, context_window FROM model_windows;

DROP TABLE model_windows;
