# Infrastructure notes

- **PostgreSQL + API:** use the root [`docker-compose.yml`](../docker-compose.yml) (Postgres + AffilFlow API).
- **Authentication** is handled in the Go API: Google/Facebook OAuth and AffilFlow-issued JWTs — see [`docs/04-authentication-keycloak.md`](../docs/04-authentication-keycloak.md) (historical filename; content describes the current backend auth).
- **Hyperledger Fabric (optional):** see [`fabric/README.md`](fabric/README.md).

The former Keycloak dev stack has been removed in favor of backend-native auth.
