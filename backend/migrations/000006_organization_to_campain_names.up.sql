-- Rename legacy organization naming to campain naming.

ALTER TABLE IF EXISTS organizations RENAME TO campains;

ALTER TABLE IF EXISTS subscriptions RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS users RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS affiliate_invites RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS affiliates RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS orders RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS shopify_stores RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS woocommerce_stores RENAME COLUMN organization_id TO campain_id;
ALTER TABLE IF EXISTS affiliate_applications RENAME COLUMN organization_id TO campain_id;

ALTER INDEX IF EXISTS idx_subscriptions_org RENAME TO idx_subscriptions_campain;
ALTER INDEX IF EXISTS idx_affiliate_invites_org RENAME TO idx_affiliate_invites_campain;
ALTER INDEX IF EXISTS idx_affiliates_org RENAME TO idx_affiliates_campain;
ALTER INDEX IF EXISTS idx_affiliate_applications_org RENAME TO idx_affiliate_applications_campain;
