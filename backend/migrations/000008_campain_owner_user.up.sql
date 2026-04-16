-- Merchant ownership of campains (many campains per merchant user).
ALTER TABLE campains
    ADD COLUMN IF NOT EXISTS owner_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_campains_owner_user ON campains(owner_user_id);

-- Backfill: merchant users' linked campain becomes owned by them.
UPDATE campains c
SET owner_user_id = u.id
FROM users u
WHERE u.campain_id = c.id
  AND u.role = 'merchant'
  AND c.owner_user_id IS NULL;
