# AffilFlow

Affiliate marketing platform: **Go/Fiber** API, **Next.js** frontend with **shadcn/ui**, PostgreSQL, **backend OAuth (Google/Facebook) + AffilFlow JWTs**, Shopify/WooCommerce, **SaaS subscriptions** (free tier + €10/€20/€50), Stripe (Billing + Connect) / PayPal, optional Hyperledger Fabric.

## Repository layout

| Path | Purpose |
|------|---------|
| **[backend/](backend/)** | Go module (`go.work` member): Fiber API, `cmd/`, `internal/`, `pkg/`, `migrations/`, `apidocs/` |
| **[web/](web/)** | Next.js App Router UI (shadcn/ui): invites, directory, join landing |
| **[docs/](docs/)** | Architecture and product documentation (Markdown) |
| **[go.work](go.work)** | Workspace file so tools resolve `./backend` from the repo root |
| **[docker-compose.yml](docker-compose.yml)** | Postgres + API — see [infra/README.md](infra/README.md) |
| **[infra/fabric/README.md](infra/fabric/README.md)** | Run Hyperledger Fabric in Docker (fabric-samples test-network) |
| **[Makefile](Makefile)** | Root targets, e.g. `make fabric-up` / `make fabric-down` |

## Documentation

Full architecture, data model, per-feature flows (with diagrams), and infrastructure notes live in **[docs/](docs/README.md)**.

Start with [docs/01-project-overview.md](docs/01-project-overview.md) and [docs/02-system-architecture.md](docs/02-system-architecture.md).

Auth details: [docs/04-authentication-keycloak.md](docs/04-authentication-keycloak.md) (filename is legacy; content is current backend auth).

## API (Go) — quick start

Requirements: **Go 1.23+** and a running **PostgreSQL** instance.

```bash
cp backend/.env.example backend/.env
# Set DATABASE_URL and AUTH_JWT_SECRET (or use JWT_SKIP_VALIDATION=true only for local dev)
export AUTO_MIGRATE=true
cd backend && make run
```

`make run` executes `go run ./cmd` inside **`backend/`** (see [backend/README.md](backend/README.md)).

Migrations add AffilFlow tables to the database named in `DATABASE_URL`. Use a dedicated DB if you share the server with other apps.

- Root: `GET http://localhost:8080/` (JSON with Swagger link)
- **Swagger UI:** `http://localhost:8080/swagger/index.html` (regenerate: `make swagger` from `backend/`)
- Health: `GET http://localhost:8080/health`
- API ping: `GET http://localhost:8080/api/v1/ping`
- OAuth: `GET http://localhost:8080/api/v1/auth/providers/google/start` (and `/callback` after provider redirect)
- Protected: `GET http://localhost:8080/api/v1/auth/me` (Bearer JWT, or use `JWT_SKIP_VALIDATION=true` for dev)
- Referral redirect: `GET http://localhost:8080/ref/{code}` (requires a seeded `affiliates` row)

Build binary: `cd backend && make build` → `backend/bin/affilflow`.

**Docker image:** build context must be `backend/`:

```bash
docker build -f backend/Dockerfile -t affilflow-api backend
```

Set `DATABASE_URL`, `AUTH_JWT_SECRET`, `AUTO_MIGRATE`, OAuth env vars, etc. at runtime.

### Go workflow

- **VS Code / Cursor:** [`.vscode/settings.json`](.vscode/settings.json) — format on save + organize imports (install the **Go** extension). Open the **repo root** so `go.work` is picked up.
- **CLI:** `cd backend && make fmt` → `make vet` → `make check` (fmt + vet + tests) before you commit. From repo root, `go test ./backend/...` also works (via `go.work`).
- **API docs:** after route/handler changes, `cd backend && make swagger` and commit updated `backend/apidocs/`.

## Web (Next.js) — how to run

Requirements: **Node.js 20+** and **npm**.

1. **Start the API** on port 8080 (see above) so the browser can reach it. If you use another host or port, set `NEXT_PUBLIC_API_URL` accordingly.
2. From the **repo root**:

```bash
cd web
cp .env.example .env.local
npm install
npm run dev
```

3. Open **http://localhost:3001** (Next.js dev server; Turbopack — port set in `web/package.json` to avoid clashing with Grafana on 3000).

**Environment:** copy [web/.env.example](web/.env.example) to **`.env.local`**. Important: **`NEXT_PUBLIC_API_URL`** (defaults to `http://localhost:8080`) — the Fiber API base URL used by `fetch` calls.

**Auth:** sign in with **Continue with Google/Facebook** on `/login` or `/register`. The API redirects back to `/auth/callback?token=...`; the SPA stores the JWT and uses it as `Authorization: Bearer` for `/api/v1/*`. For quick API testing you can still paste a JWT in the **API token (dev)** banner on dashboards.

**Production build** (static optimization + `next start`):

```bash
cd web
npm run build
npm run start
```

By default **`next start`** listens on **http://localhost:3001** (see `web/package.json`).

**Docker (Postgres + API):** from the repo root, `docker compose up -d --build` starts Postgres (5432) and the API on 8080. Set `OAUTH_*` and `AUTH_JWT_SECRET` in `docker-compose.yml` or override env for real OAuth. Run the Next.js app on the host as above.

### Hyperledger Fabric (Docker)

AffilFlow does not ship a custom Fabric stack. Use the official **[fabric-samples](https://github.com/hyperledger/fabric-samples) test-network** (Docker Compose: orderer, peers, CAs).

1. Install Fabric Docker images/samples once: [Hyperledger Fabric install](https://hyperledger-fabric.readthedocs.io/en/latest/install.html).
2. Clone `fabric-samples` (e.g. next to this repo) or set **`FABRIC_SAMPLES_DIR`** to your clone path.
3. From the **repo root**: **`make fabric-up`** (runs `./network.sh up createChannel -ca` in `test-network`). Tear down with **`make fabric-down`**.

Full notes, connection profile hints, and API-in-Docker networking: **[infra/fabric/README.md](infra/fabric/README.md)**.

## Implementation status

- **Backend:** Fiber API with pgx, migrations, AffilFlow JWT + `admin` RBAC, OAuth (Google/Facebook), referral redirect, Shopify/Woo webhooks (async order processing), Stripe Billing webhook, invites + directory/apply, Stripe Connect / PayPal payout batch, optional Fabric noop, CORS for the web app.
- **Web:** Next.js + shadcn — OAuth login, admin invite (copy link), program directory + apply, `/join/{token}` flow.
- **Ops:** `backend/Dockerfile`, root `docker-compose.yml` (Postgres + API).

See [docs/](docs/README.md) for the full roadmap.
