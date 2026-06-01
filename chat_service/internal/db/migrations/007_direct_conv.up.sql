ALTER TABLE rooms
    ADD COLUMN type VARCHAR(10) NOT NULL DEFAULT 'group'
        CHECK (type IN ('group', 'direct')),
    ALTER COLUMN name DROP NOT NULL;

CREATE TABLE IF NOT EXISTS direct_conversations (
    room_id UUID PRIMARY KEY REFERENCES rooms(id) ON DELETE CASCADE,
    user1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    CHECK (user1_id < user2_id),
    UNIQUE (user1_id, user2_id)
);

CREATE INDEX IF NOT EXISTS idx_direct_conv_user1 ON direct_conversations(user1_id);
CREATE INDEX IF NOT EXISTS idx_direct_conv_user2 ON direct_conversations(user2_id);
