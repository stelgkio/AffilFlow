-- Per-merchant WooCommerce webhook signing secret (paste into WooCommerce → Webhooks → Secret).
ALTER TABLE woocommerce_stores
    ADD COLUMN IF NOT EXISTS webhook_secret TEXT;
