CREATE TABLE IF NOT EXISTS friendships (
    requester_id UUID REFERENCES users(id) ON DELETE CASCADE,
    addressee_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(10) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(requester_id, addressee_id),
    CHECK(requester_id <> addressee_id),
    CHECK(status IN ('pending', 'accepted', 'cancelled', 'declined'))
);

CREATE INDEX idx_friendships_addressee ON friendships(addressee_id);
CREATE INDEX idx_friendships_requester ON friendships(requester_id);
CREATE UNIQUE INDEX idx_friendships_unique_pair ON friendships(LEAST(requester_id::text, addressee_id::text), GREATEST(requester_id::text, addressee_id::text));                                                                                                                                 