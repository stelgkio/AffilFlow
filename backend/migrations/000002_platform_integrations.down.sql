DROP TABLE IF EXISTS affiliate_applications;
DROP TABLE IF EXISTS woocommerce_stores;
DROP TABLE IF EXISTS shopify_stores;

ALTER TABLE affiliates DROP COLUMN IF EXISTS paypal_email;
ALTER TABLE affiliates DROP COLUMN IF EXISTS stripe_connect_account_id;

ALTER TABLE campains DROP COLUMN IF EXISTS stripe_customer_id;
