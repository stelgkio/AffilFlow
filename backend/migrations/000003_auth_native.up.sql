-- Native auth (OAuth + AffilFlow JWT). Replaces external IdP subject-only users.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'affiliate'
        CHECK (role IN ('affiliate', 'admin')),
    ADD COLUMN IF NOT EXISTS display_name TEXT;

COMMENT ON COLUMN users.role IS 'Realm-equivalent role: affiliate (default) or admin (merchant staff).';
COMMENT ON COLUMN users.display_name IS 'Display name from OAuth profile or local.';

CREATE TABLE IF NOT EXISTS auth_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL CHECK (provider IN ('google', 'facebook')),
    provider_user_id TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT,
    profile JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_identities_user_id ON auth_identities(user_id);
