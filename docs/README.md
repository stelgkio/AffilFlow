# AffilFlow — Technical documentation

This folder describes the **AffilFlow** affiliate marketing platform: **backend** (Go/Fiber), **frontend** (Next.js + shadcn/ui), data model, per-feature behavior, sequence diagrams, and infrastructure. Use it as the single source of truth before and alongside implementation.

| Document | Contents |
|----------|----------|
| [01-project-overview.md](01-project-overview.md) | Vision, scope, tech stack, glossary |
| [02-system-architecture.md](02-system-architecture.md) | Layers, components, deployment view, diagrams |
| [03-data-model.md](03-data-model.md) | Entities, relationships, key fields |
| [04-authentication-keycloak.md](04-authentication-keycloak.md) | JWT/OIDC flow, roles, middleware |
| [05-affiliate-tracking.md](05-affiliate-tracking.md) | Referral links, clicks, cookies, redirect rules |
| [06-orders-commissions.md](06-orders-commissions.md) | Order lifecycle, commission calculation, statuses |
| [07-webhooks-shopify-woocommerce.md](07-webhooks-shopify-woocommerce.md) | Verification, payload mapping, shared processing |
| [08-payments-payouts.md](08-payments-payouts.md) | Stripe, PayPal, payout batch, reconciliation |
| [09-blockchain-hyperledger-fabric.md](09-blockchain-hyperledger-fabric.md) | Chain writes, async, retries, dev network |
| [10-infrastructure-docker.md](10-infrastructure-docker.md) | Postgres, Keycloak, Fabric, networking, env vars |
| [11-frontend-nextjs-shadcn.md](11-frontend-nextjs-shadcn.md) | Next.js app, shadcn/ui, Keycloak in the browser, API integration |
| [12-platform-subscriptions-billing.md](12-platform-subscriptions-billing.md) | SaaS tiers (free 3 invites; €10/€20/€50), Stripe Billing vs affiliate payouts |
| [13-affiliate-onboarding-and-discovery.md](13-affiliate-onboarding-and-discovery.md) | Company invites (email + link for chat), affiliate directory / apply |

**Reading order:** 01 → 02 → 03, then **11**, **12**, or **13** (product/onboarding), then any feature doc (04–09) as needed, then 10.
