CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    slug             TEXT NOT NULL UNIQUE,
    type             TEXT NOT NULL DEFAULT '',
    description      TEXT NOT NULL DEFAULT '',
    date             TEXT NOT NULL DEFAULT '',
    time             TEXT NOT NULL DEFAULT '',
    location_name    TEXT NOT NULL DEFAULT '',
    address          TEXT NOT NULL DEFAULT '',
    cover_image_url  TEXT NOT NULL DEFAULT '',
    host_message     TEXT NOT NULL DEFAULT '',
    theme            TEXT NOT NULL DEFAULT '',
    primary_color    TEXT NOT NULL DEFAULT '',
    secondary_color  TEXT NOT NULL DEFAULT '',
    background_color TEXT NOT NULL DEFAULT '',
    text_color       TEXT NOT NULL DEFAULT '',
    pix_key          TEXT NOT NULL DEFAULT '',
    pix_holder_name  TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'draft',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE guests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    phone            TEXT NOT NULL DEFAULT '',
    invite_code      TEXT NOT NULL UNIQUE,
    qr_code_token    TEXT NOT NULL UNIQUE,
    max_companions   INT NOT NULL DEFAULT 0,
    rsvp_status      TEXT NOT NULL DEFAULT 'pending',
    checked_in_at    TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rsvps (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_id         UUID NOT NULL REFERENCES guests(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'pending',
    companions_count INT NOT NULL DEFAULT 0,
    message          TEXT NOT NULL DEFAULT '',
    responded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (guest_id)
);

CREATE TABLE gifts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    image_url        TEXT NOT NULL DEFAULT '',
    value_cents      INT,
    external_link    TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'available',
    allow_reservation BOOLEAN NOT NULL DEFAULT TRUE,
    allow_pix        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE gift_transactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gift_id       UUID NOT NULL REFERENCES gifts(id) ON DELETE CASCADE,
    event_id      UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    guest_name    TEXT NOT NULL,
    guest_contact TEXT NOT NULL DEFAULT '',
    type          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    message       TEXT NOT NULL DEFAULT '',
    confirmed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
