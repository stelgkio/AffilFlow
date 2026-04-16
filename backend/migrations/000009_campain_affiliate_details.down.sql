ALTER TABLE campains
    DROP COLUMN IF EXISTS attribution_window_days,
    DROP COLUMN IF EXISTS default_commission_rate,
    DROP COLUMN IF EXISTS terms_url,
    DROP COLUMN IF EXISTS brand_website_url,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS tagline;
