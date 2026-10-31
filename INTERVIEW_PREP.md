# Interview Guide: Cross-Team Knowledge Relay

## 60-second overview

This is a portfolio prototype for helping engineers find expertise outside their immediate team. A user posts a request with skills and a team. The Go/Gin API validates it, asks Neo4j for cross-team candidates, and scores candidates using skill proficiency and skill recency. It stores the request and its matches in PostgreSQL. Redis coordinates idempotent retries, while PostgreSQL keeps the durable replay record. The current flow is synchronous and uses synthetic profiles; it does not process payments or claim production-scale performance.

## Explain one request end to end

1. The Next.js form sends `POST /api/v1/bounties` with a stable `X-Idempotency-Key`.
2. The API validates the payload and removes blank or duplicate skills.
3. Redis `SET NX` claims a short-lived processing lock. A completed result is cached for 24 hours; a key reused with a different normalized payload receives `409 Conflict`. A simultaneous same-payload retry gets an in-progress `409` and can retry.
4. Neo4j matches requested skills and filters out experts on the poster's team. Scores combine normalized skill level and exponential activity recency; the top three are returned.
5. A single PostgreSQL transaction saves both the bounty and the idempotency response. If the Redis cache is later lost, PostgreSQL can still replay the original result.
6. The frontend displays the response and reuses the same key if it needs to retry. The feed marks past-deadline open requests as expired when it is loaded.

## Engineering decisions worth discussing

### Why use three data stores?

- **Neo4j** models engineer-to-skill and engineer-to-team relationships, which are the paths used for matching.
- **PostgreSQL** stores request records and the durable idempotency result in one transaction.
- **Redis** provides a 30-second feed cache and atomic `SET NX` coordination plus a 24-hour response cache for idempotent retries.

PostgreSQL remains the source of truth. Redis alone cannot provide an atomic transaction spanning Redis and PostgreSQL, so durable duplicate protection is stored in PostgreSQL with the bounty.

The feed cache uses a revisioned key so a newly saved bounty invalidates earlier feed data. Its TTL is capped by the next open bounty expiry. This avoids repeated database reads for closely spaced feed requests, but is not a measured latency guarantee.

### What does the score mean?

For each matching skill, the score adds `0.6 * (skill level / 5) + 0.4 * exp(-0.03 * days since use)`, then adds `reputation / 5`. It excludes same-team candidates, orders by score, and limits results to three. Scores are not normalized by the number of requested skills, so experts matching more skills can accumulate a higher score. This is a demo heuristic—not a learned model, PageRank implementation, or measured relevance metric.

### How is the graph data created?

`backend/scripts/seed.go` deterministically generates synthetic engineer/team/skill data. Defaults are 10,000 engineer IDs and five skill edges each. The script makes the dataset reproducible; it does not prove a benchmark result or mean a deployed database has been seeded.

### What does idempotency guarantee?

Within the API, requests with the same key and same normalized payload replay the original response instead of inserting a second bounty. Reusing the key for a different payload is rejected. Redis coordinates the common path; PostgreSQL's transaction stores the bounty and replay record together. This does not make an external payment or cross-database transaction atomic.

## Be direct about current limitations

- There is no authentication or authorization.
- The deployment has not been measured under load. There is no evidence for sub-200 ms routing or a 35% improvement, so do not claim those numbers.
- The feed cache is short-lived and only optimizes repeat reads; it does not eliminate cold starts, network latency, or the synchronous matching/database work on new requests.
- Matching runs synchronously. There is no queue, Redis Pub/Sub, WebSocket server, or real-time notification fanout.
- The escrow/payout workflow and actual payment processing are not implemented.
- The Neo4j seed utility can generate 10,000 profiles and 50,000 skill relationships, but those counts should only be described as a loaded test dataset after actually running it and verifying database counts.
- The current integration test requires PostgreSQL and Redis; it skips when test-service URLs are absent.

## Strong, accurate resume bullets

- Built a Go/Gin API for cross-team engineering requests, persisting bounties in PostgreSQL and ranking cross-team experts from a Neo4j skill graph using proficiency and activity recency.
- Added Redis-backed idempotency and short-lived feed caching with payload-conflict detection and durable PostgreSQL response replay; implemented a deterministic Neo4j seed utility configured for 10,000 synthetic engineers and 50,000 skill edges.

Only add dataset counts, latency, throughput, or percentage improvements after running and retaining reproducible measurements.

## Questions to practice

1. What happens if the client times out after the database commit?
2. Why is the idempotency record written in the same PostgreSQL transaction as the bounty?
3. What happens if Redis is unavailable before the request starts? After the PostgreSQL commit?
4. How would you make matching asynchronous without losing a request between Postgres and a queue?
5. How would you test score quality instead of only testing that the query returns candidates?
6. What additional authentication, authorization, privacy, and abuse controls would an internal employee directory require?
