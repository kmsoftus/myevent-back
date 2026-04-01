ALTER TABLE guests ADD COLUMN short_code TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX guests_short_code_event_idx ON guests (event_id, short_code) WHERE short_code <> '';
