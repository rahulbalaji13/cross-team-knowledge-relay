# Cross-Team Knowledge Relay

A portfolio prototype for routing engineering help requests to relevant experts outside the requester's team. The current implementation is a synchronous, end-to-end slice: the Go API validates a request, Neo4j ranks sample experts, PostgreSQL stores the request and its result, and Redis deduplicates retries.

## What is implemented

- **API:** Go 1.22 and Gin, with request validation, CORS, and a health endpoint.
- **Matching:** Neo4j Cypher matches required skills to engineer profiles, excludes the poster's team, and ranks candidates using skill level and activity recency.
- **Persistence:** PostgreSQL stores bounties, matched experts, and durable idempotency records.
- **Retry safety:** `X-Idempotency-Key` uses Redis `SET NX` with a 24-hour cache. A PostgreSQL transaction stores the request hash and response with the bounty, so a retry can recover the original result even if Redis loses its cache.
- **Feed reads:** Redis caches the serialized feed for up to 30 seconds. A revision key invalidates prior entries after a new bounty is saved, and the cache lifetime is bounded by the next open bounty expiry.
- **Synthetic graph data:** `backend/scripts/seed.go` deterministically creates 10,000 synthetic engineers, each with five skill relationships by default (50,000 relationships). This is a generator; it does not mean those records have already been loaded in a hosted database.
- **Frontend:** Next.js form and feed call the API. The client keeps the same idempotency key when retrying a failed submission.

## Request flow

1. The browser sends a bounty payload and an idempotency key.
2. The API validates and normalizes the payload, then claims the key in Redis.
3. If this key already has a stored result, the API replays it. Reusing the key for a different payload returns `409 Conflict`.
4. Neo4j returns up to three cross-team expert matches.
5. In one PostgreSQL transaction, the API stores the bounty and the replayable idempotency result.
6. The API caches the result in Redis and returns `201 Created`; repeated requests with the same key return the same bounty.

PostgreSQL is the durable source of truth for both the bounty and request replay. Redis is the fast idempotency cache/coordination layer, not the only protection against duplicate writes.

## Run locally

Requirements: Docker Compose, Go 1.22+ (for local Go commands), and Node.js 20+.

```powershell
docker compose build backend
docker compose up -d postgres redis neo4j
docker compose run --rm --entrypoint /usr/local/bin/seed backend
docker compose up -d backend
```

The seed command generates 10,000 synthetic profiles and 50,000 skill relationships. To use a smaller dataset, pass values such as `-e ENGINEER_COUNT=1000 -e SKILLS_PER_ENGINEER=5` to `docker compose run`.

In another terminal:

```powershell
cd frontend
npm ci
npm run dev
```

The frontend defaults to the deployed demo API. For local work, create `frontend/.env.local` with:

```dotenv
NEXT_PUBLIC_API_URL=http://localhost:8080
```

API health: `http://localhost:8080/api/v1/health`. Neo4j Browser: `http://localhost:7474` (`neo4j` / `password` in this local-only Compose setup).

## API

### `POST /api/v1/bounties`

Required header: `X-Idempotency-Key` (up to 256 characters).

```json
{
  "title": "Need help with API latency",
  "description": "Latency increased after a recent change.",
  "skills": ["Go", "PostgreSQL"],
  "bounty_amount": 150,
  "ttl_seconds": 3600,
  "poster_team": "Commerce"
}
```

Returns `{ "status", "message", "bounty" }`, preserving the existing frontend response envelope. A successful first request returns `201`; a replay returns `200`. The same key with a different normalized payload returns `409`. A concurrent retry while the original request is still running receives `409` and can retry with the same key after it completes.

### `GET /api/v1/bounties`

Returns the latest 100 persisted requests and marks open requests past their expiration time as `EXPIRED`. Repeated reads can use the Redis feed cache; a new bounty advances the cache revision so later reads use fresh data.

## Tests

```powershell
cd backend
go test ./...
```

The persistence integration test is skipped unless `TEST_POSTGRES_URL` and `TEST_REDIS_URL` are set. The Neo4j ranking integration test is skipped unless `TEST_NEO4J_URI`, `TEST_NEO4J_USERNAME`, and `TEST_NEO4J_PASSWORD` are set. Use a disposable test database/graph because these tests create tables and temporary graph fixtures.

```powershell
$env:TEST_POSTGRES_URL = "postgres://user:pass@localhost:5432/db?sslmode=disable"
$env:TEST_REDIS_URL = "redis://localhost:6379/0"
$env:TEST_NEO4J_URI = "bolt://localhost:7687"
$env:TEST_NEO4J_USERNAME = "neo4j"
$env:TEST_NEO4J_PASSWORD = "password"
go test ./...
```

## Honest scope and limitations

This is a demo, not a production marketplace. It has no authentication/authorization, payment processing, escrow/payout workflow, Redis Pub/Sub, WebSocket server, or asynchronous matching worker. Matching is synchronous. The short feed-cache TTL is an implementation optimization, not evidence of a latency target: no latency or percentage-improvement benchmark has been run, so do not claim a sub-200 ms SLA or a 35% improvement without reproducible measurements.

## Resume wording grounded in this code

> Built a Go/Gin API for cross-team engineering help requests, persisting bounties in PostgreSQL and ranking experts from a Neo4j skill graph using proficiency and activity recency.
>
> Added Redis-backed idempotency and short-lived feed caching, with payload-conflict detection and PostgreSQL-backed result replay; created a deterministic Neo4j seed utility configured to generate 10,000 synthetic engineers and 50,000 skill edges.
