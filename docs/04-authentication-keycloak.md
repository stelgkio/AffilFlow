# 04 — Authentication (AffilFlow API)

> **Note:** This document replaces the older Keycloak-only design. The API now issues its own JWTs and completes OAuth with Google/Facebook on the backend.

## Overview

1. **OAuth** — `GET /api/v1/auth/providers/{google|facebook}/start` redirects to the provider; `GET .../callback` exchanges the code, upserts `users` + `auth_identities`, and redirects the browser to `PUBLIC_APP_BASE_URL` with `?token=<JWT>` (or to a validated `next` URL from OAuth state).
2. **API access** — Clients send `Authorization: Bearer <JWT>`. Middleware verifies HS256 using `AUTH_JWT_SECRET` and checks `iss` / `aud`.
3. **Roles** — JWT claim `roles` is an array of strings (`affiliate` default, `admin` for merchants). Routes use `RequireRoles("admin")` where needed.
4. **Company onboarding** — `POST /api/v1/onboarding/company` (authenticated) creates an organization, subscribes the free plan, sets `users.organization_id`, and promotes the user to `admin`.

## Environment (API)

| Variable | Purpose |
|----------|---------|
| `AUTH_JWT_SECRET` | HMAC secret for signing/verifying API JWTs |
| `AUTH_JWT_ISSUER` | `iss` claim (default `affilflow`) |
| `AUTH_JWT_AUDIENCE` | `aud` claim (default `affilflow-api`) |
| `AUTH_PUBLIC_BASE_URL` | Public base of this API (OAuth `redirect_uri` host) |
| `OAUTH_GOOGLE_*` / `OAUTH_FACEBOOK_*` | OAuth client credentials |
| `PUBLIC_APP_BASE_URL` | Web app origin for default post-login redirect |
| `REDIRECT_ALLOW_HOSTS` | Allowlisted hosts for `next` in OAuth state |
| `AUTH_BOOTSTRAP_ADMIN_EMAIL` | Optional: first login with this email becomes `admin` |
| `JWT_SKIP_VALIDATION` | Dev only: skip JWT verification (do not use in production) |

## Web app

- Store the JWT from the OAuth redirect query in `localStorage` (key `affilflow_jwt`) and send it as `Authorization: Bearer` to `/api/v1/*`.
- Configure `NEXT_PUBLIC_API_URL` to point at the Go API.

## Legacy Keycloak users

Users previously keyed by Keycloak `sub` remain in `users.id` as opaque strings. New OAuth users get UUID `id` values. Linking is by **email** on first OAuth login when an existing row matches.
