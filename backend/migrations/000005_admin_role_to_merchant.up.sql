-- Rename merchant staff role from admin -> merchant.

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check;

UPDATE users
SET role = 'merchant'
WHERE role = 'admin';

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('affiliate', 'merchant'));

COMMENT ON COLUMN users.role IS 'Realm-equivalent role: affiliate (default) or merchant (merchant staff).';
