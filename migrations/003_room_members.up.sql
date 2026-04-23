CREATE TABLE IF NOT EXISTS room_members (
    room_id UUID REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, room_id)
);

CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON room_members(user_id);                                                                                                                                                                                             
CREATE INDEX IF NOT EXISTS idx_room_members_room_id ON room_members(room_id);                                                                                                                                                                                   