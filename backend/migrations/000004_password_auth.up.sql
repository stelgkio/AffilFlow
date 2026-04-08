-- Email/password auth: optional bcrypt hash on users (OAuth-only users leave this NULL).

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT;

COMMENT ON COLUMN users.password_hash IS 'bcrypt hash for email/password sign-in; NULL if OAuth-only.';
