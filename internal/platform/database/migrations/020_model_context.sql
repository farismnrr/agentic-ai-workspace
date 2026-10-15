-- How much each model can hold at once, as chosen by the person. A model without a
-- row uses the built-in default, so most models never get one.
CREATE TABLE model_settings (
    provider_id    TEXT    NOT NULL REFERENCES providers (id) ON DELETE CASCADE,
    model          TEXT    NOT NULL,
    context_window INTEGER NOT NULL CHECK (context_window > 0),
    PRIMARY KEY (provider_id, model)
);
