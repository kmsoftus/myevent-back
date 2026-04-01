DROP INDEX IF EXISTS guests_short_code_event_idx;

ALTER TABLE guests DROP COLUMN IF EXISTS short_code;
