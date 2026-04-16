# 03 — Data model

## Multi-tenancy and subscriptions (platform SaaS)

Each **merchant** using AffilFlow belongs to a **campain** with a **subscription** and an **invite cap** (how many affiliates they may onboard). Defaults: **free = 3 invites**; paid **€10 / €20 / €50** per month with higher caps (see [12-platform-subscriptions-billing.md](12-platform-subscriptions-billing.md)).

```mermaid
erDiagram
  campains ||--o{ users : "members"
  campains ||--o{ affiliates : "owns"
  subscription_plans ||--o{ subscriptions : "defines"
  campains ||--o| subscriptions : "has"

  campains {
    uuid id PK
    text name
    timestamptz created_at
  }

  subscription_plans {
    text plan_key PK "free | starter | growth | scale"
    int price_eur_cents "0 for free"
    int max_invites
    text stripe_price_id
  }

  subscriptions {
    uuid id PK
    uuid campain_id FK
    text plan_key FK
    text stripe_subscription_id
    text status "active | past_due | canceled"
    timestamptz current_period_end
  }
```

Affiliate and order rows should be **scoped by `campain_id`** (add FK where missing in implementation).

### Affiliate invites and discovery (see [13](13-affiliate-onboarding-and-discovery.md))

```mermaid
erDiagram
  campains ||--o{ affiliate_invites : "issues"
  affiliate_invites {
    uuid id PK
    uuid campain_id FK
    text email "nullable"
    text token_hash UK
    timestamptz expires_at
    text status "pending | accepted | revoked"
    timestamptz created_at
  }
```

Optional **campain** columns: `slug`, `discovery_enabled`, `approval_mode` for the public directory and apply flows.

## Entity-relationship (affiliate core)

```mermaid
erDiagram
  campains ||--o{ orders : "contains"
  users ||--o{ affiliates : "owns"
  affiliates ||--o{ referrals : "generates"
  affiliates ||--o{ orders : "credited"
  affiliates ||--o{ commissions : "earns"
  affiliates ||--o{ payouts : "receives"
  orders ||--o{ commissions : "produces"
  referrals ||--o{ orders : "may attribute"

  users {
    text id PK "Keycloak sub"
    text email
    timestamptz created_at
    timestamptz updated_at
  }

  affiliates {
    uuid id PK
    uuid campain_id FK
    text user_id FK
    text code UK "referral slug"
    numeric commission_rate
    text status
    timestamptz created_at
    timestamptz updated_at
  }

  referrals {
    uuid id PK
    uuid affiliate_id FK
    text code
    inet ip
    text user_agent
    timestamptz created_at
  }

  orders {
    uuid id PK
    uuid campain_id FK
    text external_id "Shopify/Woo id"
    text source "shopify | woocommerce"
    text customer_ref
    bigint total_cents
    text currency
    uuid referral_id FK "nullable"
    uuid affiliate_id FK "nullable"
    jsonb raw_payload
    timestamptz created_at
    timestamptz updated_at
  }

  commissions {
    uuid id PK
    uuid affiliate_id FK
    uuid order_id FK
    bigint amount_cents
    text status "pending | approved | paid"
    timestamptz created_at
    timestamptz updated_at
  }

  payouts {
    uuid id PK
    uuid affiliate_id FK
    bigint total_cents
    text provider "stripe | paypal"
    text external_payout_id
    text status
    timestamptz created_at
  }
```

## Table purposes

| Table | Purpose |
|-------|---------|
| **campains** | Paying tenant; subscription and invite limits apply here |
| **subscription_plans** | Plan keys, EUR price, **max_invites**, Stripe price ids |
| **subscriptions** | Active Stripe subscription per campain |
| **users** | Mirror of Keycloak subjects; may belong to a **campain** |
| **affiliates** | Business entity: code, rate, status; links to `users` and **campain** |
| **referrals** | Immutable click stream for analytics and attribution debugging |
| **orders** | Normalized order from any source; `external_id` + `source` uniqueness enforced |
| **commissions** | Monetary obligation per order; drives payout batches |
| **payouts** | Batch or per-affiliate payout records tied to Stripe/PayPal references |

## Key constraints

- **Unique affiliate code** — prevents duplicate public URLs.
- **Unique (external_id, source)** on orders — webhook retries must not create duplicate business rows (use upsert or check-then-insert in a transaction).
- **Commission status** — `pending` → `approved` (optional manual step) → `paid` after successful payout.

## Money representation

- Store amounts as **integer cents** (`bigint`) with an ISO **currency** field on orders to avoid floating-point errors.
