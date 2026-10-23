# Deployment guide

This repository can be deployed as two services: the Go API plus PostgreSQL, and the Next.js frontend. This guide describes a secure-by-default **preview deployment**. A public write-enabled launch still requires user authentication and abuse protection.

## Backend service

Deploy the `backend` directory with the provided Dockerfile. Attach a managed PostgreSQL database and configure:

- `PORT`: usually supplied by the platform.
- `DATABASE_URL`: private connection string for PostgreSQL.
- `API_BEARER_TOKEN`: long random secret; store it only in the backend and frontend server environment.
- `ALLOWED_ORIGINS`: exact web origin(s), comma-separated. Example: `https://your-app.example.com`.

The API performs a database readiness check and creates the bounty table/indexes at startup. For production schema lifecycle management, replace startup DDL with versioned migrations before multiple API replicas are deployed.

## Frontend service

Deploy the `frontend` directory as a Next.js application. Configure server-only environment variables:

- `RELAY_API_URL`: backend base URL, with no trailing slash.
- `RELAY_API_TOKEN`: same secret as backend `API_BEARER_TOKEN`.

Do not prefix these variables with `NEXT_PUBLIC_`; the browser must never receive the API token. The Next.js routes proxy requests to the Go API, including the SSE stream.

## Local setup

```sh
docker compose up --build
```

Then create `frontend/.env.local`:

```env
RELAY_API_URL=http://localhost:8080
RELAY_API_TOKEN=local-only-change-this-token
```

From `frontend`, run `npm ci` and `npm run dev`. The UI will be available at `http://localhost:3000`.

## Before enabling public writes

The current proxy allows anonymous browser requests to use the server-side API token. The token stays secret, but the write capability is effectively public. Before enabling public writes, add a real user identity flow (for example GitHub OAuth), authorization and ownership rules, CSRF/origin checks, durable rate limiting or a managed WAF, abuse reporting/moderation, and audit logging. Do not use this prototype for actual employee personal data or payments.

Also add CI build/type/lint checks, integration tests against PostgreSQL, structured logs/metrics, alerting, backups, and versioned database migrations. The in-process rate limiter is not shared across API replicas.
