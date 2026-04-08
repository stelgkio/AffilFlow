# 10 — Infrastructure and Docker

## Services (local development)

```mermaid
flowchart TB
  subgraph app_stack [Application stack]
    API[AffilFlow container or host binary]
    PG[(PostgreSQL - app)]
  end

  subgraph id_stack [Identity stack]
    KC[Keycloak]
    PGKC[(PostgreSQL - Keycloak)]
    KC --> PGKC
  end

  subgraph fabric_stack [Fabric dev network]
    ORD[Orderer]
    PEER[Peer]
    CA[CA]
  end

  API --> PG
  API --> KC
  API -.-> ORD
  API -.-> PEER
```

## PostgreSQL (application)

- Single database for AffilFlow tables.
- Connection string in `DATABASE_URL` (pgx-compatible), e.g. `postgres://user:pass@host:5432/affilflow?sslmode=disable` for local dev.

## Keycloak

- Run **Keycloak** with a **dedicated Postgres** instance (recommended over embedded H2 for team consistency).
- Expose HTTP(S) port for admin UI and OIDC; AffilFlow uses **`KEYCLOAK_JWKS_URL`** reachable from the API process (Docker network hostname if both in Compose).
- Bootstrap admin user via env (Keycloak version-specific variables—documented in README at implementation time).

## Hyperledger Fabric

Fabric runs in **Docker** via the official **fabric-samples** [test-network](https://github.com/hyperledger/fabric-samples/tree/main/test-network) (orderer, peers, CAs — Docker Compose under the hood).

- **AffilFlow repo:** [infra/fabric/README.md](../infra/fabric/README.md) — `make fabric-up` / `make fabric-down` (wrappers around `./network.sh`).
- Install prerequisites once: [Fabric install docs](https://hyperledger-fabric.readthedocs.io/en/latest/install.html) (Docker images + binaries + samples).
- Deploy chaincode and export **connection profile** paths into `FABRIC_NETWORK_CONFIG` (see [09-blockchain-hyperledger-fabric.md](09-blockchain-hyperledger-fabric.md)).

The API container (or host) must resolve peer/orderer DNS names as in the connection profile (often `extra_hosts` or `docker network connect` to the test-network’s Docker network).

## Application container

- **Multi-stage Dockerfile:** build Go binary, minimal runtime image, non-root user.
- **depends_on** Keycloak/Postgres with healthchecks where possible.

## Environment variables (checklist)

| Area | Examples |
|------|----------|
| Server | `SERVER_PORT`, `LOG_LEVEL` |
| DB | `DATABASE_URL` |
| Keycloak | `KEYCLOAK_ISSUER`, `KEYCLOAK_JWKS_URL`, `KEYCLOAK_AUDIENCE` |
| Referral | `REDIRECT_BASE_URL`, cookie names/TTL, redirect allowlist |
| Shopify | `SHOPIFY_WEBHOOK_SECRET` |
| WooCommerce | base URL, REST keys |
| Stripe / PayPal | keys, Connect settings |
| Fabric | `FABRIC_ENABLED`, network config path, channel, chaincode, MSP paths |

## Makefile / scripts (suggested)

| Target | Action |
|--------|--------|
| `make up` | `docker compose up -d` for app DB + Keycloak stack |
| `make migrate-up` | Run SQL migrations against `DATABASE_URL` |
| `make fabric-up` | Wrapper around test-network or second compose file |
