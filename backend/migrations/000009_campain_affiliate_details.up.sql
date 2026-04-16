-- Richer affiliate-program (campain) metadata for discovery and onboarding.
ALTER TABLE campains
    ADD COLUMN IF NOT EXISTS tagline TEXT,
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS brand_website_url TEXT,
    ADD COLUMN IF NOT EXISTS terms_url TEXT,
    ADD COLUMN IF NOT EXISTS default_commission_rate NUMERIC(8, 6) NOT NULL DEFAULT 0.1
        CHECK (default_commission_rate > 0 AND default_commission_rate <= 1),
    ADD COLUMN IF NOT EXISTS attribution_window_days INTEGER NOT NULL DEFAULT 30
        CHECK (attribution_window_days >= 1 AND attribution_window_days <= 365);
