ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check;

UPDATE users
SET role = 'admin'
WHERE role = 'merchant';

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('affiliate', 'admin'));

COMMENT ON COLUMN users.role IS 'Realm-equivalent role: affiliate (default) or admin (merchant staff).';
