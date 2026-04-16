CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    refresh_token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL
);