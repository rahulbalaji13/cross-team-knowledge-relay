# Cross-Team Knowledge Relay

A web application prototype for sharing engineering help requests across teams. The current implementation includes a Next.js dashboard, GitHub OAuth sign-in, a Go API, PostgreSQL persistence, and live feed updates over Server-Sent Events (SSE).

## Implemented

- Create and list help requests (called bounties in the UI).
- Store requests in PostgreSQL. The Go API creates the table and indexes at startup.
- Authenticate posting users with GitHub OAuth; sessions are HMAC-signed, HTTP-only, secure in production, and expire after 12 hours.
- Keep the backend bearer token server-side in Next.js. The browser calls same-origin routes.
- Send live bounty-created events to connected browsers over SSE.
- Suggest up to three experts from five seeded demo profiles using skill overlap, skill recency, and reputation.

## Known limits before production

This is a portfolio application, not a production employee directory or payments product. Expert profiles remain hard-coded; matching runs in-process; database setup uses startup DDL rather than versioned migrations; and the rate limiter is per-process, not shared across replicas. Any GitHub user who can sign in can currently post. Add an organization allowlist or moderation, distributed abuse controls/WAF, ownership and audit policy, versioned migrations, CI checks, and PostgreSQL integration tests before a broad public launch. The amounts are illustrative only; no money moves.

Architecture diagrams and the older interview guide include proposed components such as Neo4j, Redis, Kafka, and escrow. Those are design ideas and are not part of the running request path.

## Local development

Requirements: Docker Compose, Node.js, and a GitHub OAuth app.

1. Start PostgreSQL and the Go API: `docker compose up --build`.
2. Copy `frontend/.env.example` to `frontend/.env.local`.
3. Create a GitHub OAuth app with callback `http://localhost:3000/api/auth/callback`; set its client ID and secret in `frontend/.env.local`.
4. Set `SESSION_SECRET` to a random value of at least 32 characters.
5. From `frontend`, run `npm ci && npm run dev`.

The UI is available at `http://localhost:3000`. The API listens on `http://localhost:8080`.

## API

- `GET /api/v1/health` — readiness check including PostgreSQL connectivity.
- `GET /api/v1/bounties` — latest unexpired requests (maximum 100).
- `POST /api/v1/bounties` — create a request; requires the backend bearer token and authenticated creator field.
- `GET /api/v1/events` — SSE stream of new requests.

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for service configuration and the public-launch checklist.
