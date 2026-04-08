-- Enable trigram extension for fuzzy/substring search on guest names
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- events: filtering by owner
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id);

-- guests: every listing and count by event; trigram for name search
CREATE INDEX IF NOT EXISTS idx_guests_event_id ON guests (event_id);
CREATE INDEX IF NOT EXISTS idx_guests_name_trgm ON guests USING GIN (name gin_trgm_ops);

-- rsvps: listing and count by event
CREATE INDEX IF NOT EXISTS idx_rsvps_event_id ON rsvps (event_id);

-- gifts: listing and count by event
CREATE INDEX IF NOT EXISTS idx_gifts_event_id ON gifts (event_id);

-- gift_transactions: listing by event; expiry sweep by status + created_at
CREATE INDEX IF NOT EXISTS idx_gift_transactions_event_id ON gift_transactions (event_id);
CREATE INDEX IF NOT EXISTS idx_gift_transactions_gift_id ON gift_transactions (gift_id);
CREATE INDEX IF NOT EXISTS idx_gift_transactions_status_created_at ON gift_transactions (status, created_at) WHERE status = 'pending';
