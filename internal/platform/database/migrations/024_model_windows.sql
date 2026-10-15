-- Context window sizes a provider reported for its models when the key was verified. They are the default for a
-- model; a size the person chose (model_settings) wins over them.
CREATE TABLE model_windows (
    provider_id    TEXT    NOT NULL REFERENCES providers (id) ON DELETE CASCADE,
    model          TEXT    NOT NULL,
    context_window INTEGER NOT NULL CHECK (context_window > 0),
    PRIMARY KEY (provider_id, model)
);
