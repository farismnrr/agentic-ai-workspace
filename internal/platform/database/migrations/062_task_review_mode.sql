ALTER TABLE tasks ADD COLUMN review_mode TEXT NOT NULL DEFAULT 'swift'
    CHECK (review_mode IN ('swift', 'deep-review'));
