-- Dummy campaign/program data for local development.
-- Usage:
--   psql "$DATABASE_URL" -f backend/seeds/demo_campaigns.sql

INSERT INTO campains (name, slug, discovery_enabled, approval_mode)
VALUES
  ('Athena Wellness Spring Campaign', 'athena-wellness', true, 'request_to_join'),
  ('Neon Gadgets Creator Campaign', 'neon-gadgets', true, 'open'),
  ('Everlane Home Partner Campaign', 'everlane-home', true, 'invite_only'),
  ('Northstar Fitness Ambassador Campaign', 'northstar-fitness', true, 'open'),
  ('BluePeak Apparel Influencer Campaign', 'bluepeak-apparel', true, 'request_to_join')
ON CONFLICT (slug) DO UPDATE
SET
  name = EXCLUDED.name,
  discovery_enabled = EXCLUDED.discovery_enabled,
  approval_mode = EXCLUDED.approval_mode,
  updated_at = now();
