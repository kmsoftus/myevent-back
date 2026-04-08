-- Enable trigram extension for fuzzy/substring search on guest names
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- events: filtering by owner
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id);

-- replace legacy single-column indexes with composite variants used by paged listings
DROP INDEX IF EXISTS idx_guests_event_id;
DROP INDEX IF EXISTS idx_rsvps_event_id;
DROP INDEX IF EXISTS idx_gifts_event_id;
DROP INDEX IF EXISTS idx_gift_transactions_event_id;

-- guests: listing/count by event and sort by created_at; trigram for name search
CREATE INDEX IF NOT EXISTS idx_guests_event_id_created_at ON guests (event_id, created_at);
CREATE INDEX IF NOT EXISTS idx_guests_name_trgm ON guests USING GIN (name gin_trgm_ops);

-- rsvps: listing/count by event and sort by created_at
CREATE INDEX IF NOT EXISTS idx_rsvps_event_id_created_at ON rsvps (event_id, created_at);

-- gifts: listing/count by event and sort by created_at
CREATE INDEX IF NOT EXISTS idx_gifts_event_id_created_at ON gifts (event_id, created_at);

-- gift_transactions: listing by event; expiry sweep by status + created_at
CREATE INDEX IF NOT EXISTS idx_gift_transactions_event_id_created_at ON gift_transactions (event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gift_transactions_gift_id ON gift_transactions (gift_id);
CREATE INDEX IF NOT EXISTS idx_gift_transactions_status_created_at ON gift_transactions (status, created_at) WHERE status = 'pending';
