# Cross-Team Knowledge Relay

A web application prototype for sharing engineering help requests across teams. The current implementation includes a Next.js dashboard, a Go API, PostgreSQL persistence, and live feed updates over Server-Sent Events (SSE).

## Implemented

- Create and list help requests (called bounties in the UI).
- Store requests in PostgreSQL. The Go API creates the table and indexes at startup.
- Submit demo requests without an account; the frontend labels their creator as `Community`.
- Keep the backend bearer token server-side in Next.js. The browser calls same-origin routes.
- Send live bounty-created events to connected browsers over SSE.
- Suggest up to three experts from five seeded demo profiles using skill overlap, skill recency, and reputation.

## Known limits before production

This is a portfolio application, not a production employee directory or payments product. Expert profiles remain hard-coded; matching runs in-process; database setup uses startup DDL rather than versioned migrations; and the rate limiter is per-process, not shared across replicas. Posting is open in this demo and submissions are attributed to `Community`; add authentication and authorization before using it for an organization. A broad public launch still needs distributed abuse controls/WAF, ownership and retention policy, versioned migrations, and PostgreSQL integration tests. The amounts are illustrative only; no money moves.

Architecture diagrams and the older interview guide include proposed components such as Neo4j, Redis, Kafka, and escrow. Those are design ideas and are not part of the running request path.

## Local development

Requirements: Docker Compose and Node.js.

1. Start PostgreSQL and the Go API: `docker compose up --build`.
2. Copy `frontend/.env.example` to `frontend/.env.local`.
3. From `frontend`, run `npm ci && npm run dev`.

The UI is available at `http://localhost:3000`. The API listens on `http://localhost:8080`.

## API

- `GET /api/v1/health` — readiness check including PostgreSQL connectivity.
- `GET /api/v1/bounties` — latest unexpired requests (maximum 100).
- `POST /api/v1/bounties` — create a request; requires the backend bearer token and authenticated creator field.
- `GET /api/v1/events` — SSE stream of new requests.

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for service configuration and the public-launch checklist.
