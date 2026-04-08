# 05 — Affiliate tracking

## Referral URL

Shoppers receive links of the form:

```http
GET /ref/:code
```

Where `:code` matches `affiliates.code` in the database.

## Flow

```mermaid
sequenceDiagram
  participant Shopper
  participant API as AffilFlow
  participant PG as PostgreSQL

  Shopper->>API: GET /ref/SUMMER2026
  API->>PG: SELECT affiliate by code
  alt Found
    API->>PG: INSERT referral click
    API->>Shopper: Set-Cookie aff_ref=...; 302 Location
  else Not found
    API->>Shopper: 302 to default landing or 404
  end
```

## Click storage

Each request with a valid code inserts a **referrals** row:

- `affiliate_id`, `code`
- `ip`, `user_agent` (from request headers)
- `created_at`

## Cookie / session attribution

To correlate a later **checkout** with an affiliate when the e-commerce platform does not pass metadata:

- Set an **HTTP-only**, **Secure** (in production), **SameSite** cookie (policy TBD: often `Lax` for cross-site redirects) containing either the **code** or a **signed opaque token** with expiry.
- Configurable **TTL** (e.g. 30 days) matches business attribution window.

Middleware on **API routes** (or documented convention for storefront) can read this cookie so internal tools know “last touch” affiliate.

> **Note:** Shopify/Woo webhooks often do not include browser cookies. Production attribution usually combines: cookie where possible, **discount codes**, **UTM parameters**, or **manual mapping** in the webhook processor. This doc assumes the **order processor** implements the agreed rule (e.g. match customer email + time window, or metafield with affiliate id). The minimal implementation stores referral context when the platform provides it in the payload.

## Redirect safety

- **`next` query parameter:** Only allow redirects to URLs in an **allowlist** (config) to prevent open redirects.
- Default: redirect to `REDIRECT_BASE_URL` from environment.

## Code generation

When creating an affiliate:

- Generate a **short unique** alphanumeric code; retry on collision.
- Enforce **case sensitivity** policy (typically case-insensitive lookup for UX).
