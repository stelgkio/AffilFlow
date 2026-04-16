DROP INDEX IF EXISTS idx_campains_owner_user;
ALTER TABLE campains DROP COLUMN IF EXISTS owner_user_id;
