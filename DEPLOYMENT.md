# Deployment guide

The app is deployed as a Go API plus managed PostgreSQL, with a Next.js frontend. Use separate development and production GitHub OAuth apps. This repository is ready for a **staged public preview**; complete the final checklist before promoting it as a production service.

## 1. Backend and PostgreSQL

Deploy the `backend` directory using `backend/Dockerfile`. Attach a managed PostgreSQL database and set:

- `DATABASE_URL`: private PostgreSQL connection string.
- `API_BEARER_TOKEN`: a high-entropy secret shared only with the Next.js server environment.
- `ALLOWED_ORIGINS`: exact frontend origin, such as `https://relay.example.com`.
- `PORT`: use the port assigned by the host, commonly provided automatically.

The API checks database readiness at startup and creates the initial table and indexes. For multiple replicas and zero-downtime schema changes, replace startup DDL with versioned migrations before launch.

## 2. GitHub OAuth app

Create a GitHub OAuth app for the deployment. Set its authorization callback URL to:

```
https://YOUR_FRONTEND_HOST/api/auth/callback
```

The callback must match exactly. Set the following variables on the Next.js host:

- `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` from the OAuth app.
- `GITHUB_ALLOWED_USERS`: comma-separated GitHub logins allowed to post, such as `rahulbalaji13`. Set this for a controlled preview; if omitted, any GitHub user who signs in can post.
- `SESSION_SECRET`: at least 32 random characters; generate a fresh value and keep it private.
- `PUBLIC_SITE_URL`: canonical HTTPS frontend origin, with no trailing slash.
- `RELAY_API_URL`: backend HTTPS origin, with no trailing slash.
- `RELAY_API_TOKEN`: same value as backend `API_BEARER_TOKEN`.

Keep OAuth and API secrets in the hosting provider’s secret manager. Never commit them or use a `NEXT_PUBLIC_*` prefix.

## 3. Local setup

```sh
docker compose up --build
```

Copy `frontend/.env.example` to `frontend/.env.local`. Use a development OAuth app with callback `http://localhost:3000/api/auth/callback`, set the local API token to match `docker-compose.yml`, and provide a random `SESSION_SECRET`. From `frontend`, run `npm ci && npm run dev`.

## 4. Public-launch work still required

- Restrict sign-in to the intended organization or add moderation and abuse reporting. At present any GitHub account can post.
- Put distributed rate limiting and bot protection at a managed edge/WAF; the Go limiter is in-memory and per instance.
- Add versioned database migrations, backups, restore drills, and a retention policy.
- Add CI build/type/lint checks and PostgreSQL integration tests.
- Add structured logs, metrics, alerts, and operational runbooks.
- Decide data ownership/privacy rules before replacing the synthetic expert profiles with real employee data.
- Do not enable real payments without a separately designed and reviewed payment/escrow system.

The app should be treated as a staged preview until these items are addressed. No live deployment is created by these repository changes.
