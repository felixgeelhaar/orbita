CREATE TABLE oauth_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    provider TEXT NOT NULL,
    access_token BYTEA NOT NULL,
    refresh_token BYTEA,
    token_type TEXT,
    expiry TIMESTAMPTZ,
    scopes TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider)
);
