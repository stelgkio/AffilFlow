# AffilFlow API (Go)

Fiber + PostgreSQL backend. Module: `github.com/stelgkio/affilflow/backend`.

From this directory (with `.env` at repo root, run from parent shell that already loaded env, or `export` vars manually):

```bash
AUTO_MIGRATE=true go run ./cmd
```

Seed dummy public campaigns/programs:

```bash
psql "$DATABASE_URL" -f backend/seeds/demo_campaigns.sql
```

From the **repository root**: `make run` (recommended — runs this module with cwd `backend/` so `file://migrations` resolves correctly).

- **Docker:** `docker build -f backend/Dockerfile -t affilflow-api backend`
- **Swagger:** `make swagger` (root) or `make swagger` after `cd` here — regenerates `apidocs/`.
