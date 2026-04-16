-- Stripe / store linkage / applications

ALTER TABLE campains
    ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT;

CREATE TABLE IF NOT EXISTS shopify_stores (
    campain_id UUID PRIMARY KEY REFERENCES campains(id) ON DELETE CASCADE,
    shop_domain TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS woocommerce_stores (
    campain_id UUID PRIMARY KEY REFERENCES campains(id) ON DELETE CASCADE,
    site_base_url TEXT NOT NULL UNIQUE,
    webhook_secret TEXT
);

ALTER TABLE affiliates
    ADD COLUMN IF NOT EXISTS stripe_connect_account_id TEXT,
    ADD COLUMN IF NOT EXISTS paypal_email TEXT;

CREATE TABLE IF NOT EXISTS affiliate_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campain_id UUID NOT NULL REFERENCES campains(id) ON DELETE CASCADE,
    applicant_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    applicant_email TEXT,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_affiliate_applications_campain ON affiliate_applications(campain_id);
