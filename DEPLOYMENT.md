# Deployment guide

The app is deployed as a Go API plus managed PostgreSQL, with a Next.js frontend. The demo does not use GitHub OAuth. Anyone who can reach the frontend can submit a request, and requests are attributed to `Community`. Add authentication, authorization, and moderation before using it with a real organization.

## 1. Backend and PostgreSQL

Deploy the `backend` directory using `backend/Dockerfile`. Attach a managed PostgreSQL database and set:

- `DATABASE_URL`: private PostgreSQL connection string.
- `API_BEARER_TOKEN`: a high-entropy secret shared only with the Next.js server environment.
- `ALLOWED_ORIGINS`: exact frontend origin, such as `https://relay.example.com`.
- `PORT`: use the port assigned by the host, commonly provided automatically.

The API checks database readiness at startup and creates the initial table and indexes. For multiple replicas and zero-downtime schema changes, replace startup DDL with versioned migrations before launch.

## 2. Next.js frontend

Set these server-side variables on the Next.js host:

- `RELAY_API_URL`: backend HTTPS origin, with no trailing slash.
- `RELAY_API_TOKEN`: same value as backend `API_BEARER_TOKEN`.

Keep the API token in the hosting provider’s secret manager. Never commit it or use a `NEXT_PUBLIC_*` prefix. Copy `frontend/.env.example` to `frontend/.env.local` for local development.

## 3. Local setup

```sh
docker compose up --build
```

From `frontend`, run `npm ci && npm run dev`. Open the frontend at `http://localhost:3000`.

## 4. Public-launch work still required

- Add authentication, authorization, moderation, and abuse reporting before using this with real teams. Demo submissions are currently open and attributed to `Community`.
- Put distributed rate limiting and bot protection at a managed edge/WAF; the Go limiter is in-memory and per instance.
- Add versioned database migrations, backups, restore drills, and a retention policy.
- Add PostgreSQL integration tests.
- Add structured logs, metrics, alerts, and operational runbooks.
- Decide data ownership/privacy rules before replacing the synthetic expert profiles with real employee data.
- Do not enable real payments without a separately designed and reviewed payment/escrow system.

The app should be treated as a staged preview until these items are addressed. No live deployment is created by these repository changes.
