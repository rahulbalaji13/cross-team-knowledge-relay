# Cross-Team Knowledge Relay

A prototype for posting engineering help requests and surfacing relevant experts from other teams. The current application is a small, deployable demo: a Next.js UI, a Go API, PostgreSQL persistence for bounties, and Server-Sent Events (SSE) for live feed updates.

## What runs today

- Create and list help requests (called bounties in the UI).
- Persist requests in PostgreSQL; the API creates the required table and indexes at startup.
- Suggest up to three matches from five seeded demo experts using skill overlap, skill recency, and reputation.
- Stream newly created requests to connected browsers over SSE.
- Keep the API bearer token on the Next.js server; the browser calls same-origin Next.js routes.

## Current limits

This is a portfolio prototype, not a production employee directory or payment product. Expert profiles are hard-coded demo data. Matching runs in the API process. The API has a simple per-process rate limiter; it is not a distributed limiter. OAuth/user identity, tenant authorization, abuse controls, operational dashboards, and automated integration tests still need to be added before unrestricted public use. There are no real payments; the displayed amount is illustrative.

Architecture diagrams and the older interview guide include proposed components such as Neo4j, Redis, Kafka, and escrow. Those are design ideas and are not part of the running request path.

## Local development

Requirements: Docker Compose.

```sh
docker compose up --build
```

The API is available at `http://localhost:8080`. To run the frontend, set `RELAY_API_URL=http://localhost:8080` and `RELAY_API_TOKEN=local-only-change-this-token` in `frontend/.env.local`, then run `npm ci && npm run dev` from `frontend`. The API creates its table on startup.

## API

- `GET /api/v1/health` — liveness/readiness check including database connectivity.
- `GET /api/v1/bounties` — latest unexpired requests (maximum 100).
- `POST /api/v1/bounties` — create a request; requires `Authorization: Bearer <API_BEARER_TOKEN>`.
- `GET /api/v1/events` — SSE stream of newly created requests.

The public web UI uses same-origin server routes under `/api/bounties`; do not put the backend bearer token in a `NEXT_PUBLIC_*` variable.

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md). Keep the site read-only or behind an identity-aware access layer until user authentication and abuse controls are configured.
