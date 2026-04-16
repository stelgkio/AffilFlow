-- AffilFlow initial schema (campains, subscriptions, affiliate core)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Subscription catalog (seeded)
CREATE TABLE subscription_plans (
    plan_key TEXT PRIMARY KEY,
    price_eur_cents INTEGER NOT NULL DEFAULT 0,
    max_invites INTEGER NOT NULL,
    stripe_price_id TEXT
);

INSERT INTO subscription_plans (plan_key, price_eur_cents, max_invites) VALUES
    ('free', 0, 3),
    ('starter', 1000, 20),
    ('growth', 2000, 60),
    ('scale', 5000, 200);

CREATE TABLE campains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    discovery_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    approval_mode TEXT NOT NULL DEFAULT 'invite_only'
        CHECK (approval_mode IN ('open', 'request_to_join', 'invite_only')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campain_id UUID NOT NULL REFERENCES campains(id) ON DELETE CASCADE,
    plan_key TEXT NOT NULL REFERENCES subscription_plans(plan_key),
    stripe_subscription_id TEXT,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'past_due', 'canceled', 'trialing', 'incomplete')),
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_campain ON subscriptions(campain_id);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT,
    campain_id UUID REFERENCES campains(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE affiliate_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campain_id UUID NOT NULL REFERENCES campains(id) ON DELETE CASCADE,
    email TEXT,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'revoked')),
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_affiliate_invites_campain ON affiliate_invites(campain_id);

CREATE TABLE affiliates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campain_id UUID NOT NULL REFERENCES campains(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE,
    commission_rate NUMERIC(8, 6) NOT NULL DEFAULT 0.1,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'paused', 'removed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (campain_id, user_id)
);

CREATE INDEX idx_affiliates_campain ON affiliates(campain_id);

CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referrals_affiliate ON referrals(affiliate_id);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campain_id UUID NOT NULL REFERENCES campains(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('shopify', 'woocommerce')),
    customer_ref TEXT,
    total_cents BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'EUR',
    referral_id UUID REFERENCES referrals(id) ON DELETE SET NULL,
    affiliate_id UUID REFERENCES affiliates(id) ON DELETE SET NULL,
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (campain_id, external_id, source)
);

CREATE INDEX idx_orders_affiliate ON orders(affiliate_id);

CREATE TABLE commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'paid')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_id, affiliate_id)
);

CREATE INDEX idx_commissions_status ON commissions(status);

CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    affiliate_id UUID NOT NULL REFERENCES affiliates(id) ON DELETE CASCADE,
    total_cents BIGINT NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('stripe', 'paypal')),
    external_payout_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
