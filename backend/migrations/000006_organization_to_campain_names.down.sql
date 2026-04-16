ALTER TABLE IF EXISTS subscriptions RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS users RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS affiliate_invites RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS affiliates RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS orders RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS shopify_stores RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS woocommerce_stores RENAME COLUMN campain_id TO organization_id;
ALTER TABLE IF EXISTS affiliate_applications RENAME COLUMN campain_id TO organization_id;

ALTER TABLE IF EXISTS campains RENAME TO organizations;

ALTER INDEX IF EXISTS idx_subscriptions_campain RENAME TO idx_subscriptions_org;
ALTER INDEX IF EXISTS idx_affiliate_invites_campain RENAME TO idx_affiliate_invites_org;
ALTER INDEX IF EXISTS idx_affiliates_campain RENAME TO idx_affiliates_org;
ALTER INDEX IF EXISTS idx_affiliate_applications_campain RENAME TO idx_affiliate_applications_org;
