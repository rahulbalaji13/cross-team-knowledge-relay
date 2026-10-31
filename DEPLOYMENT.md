# Running the demo

## Local stack

The root `docker-compose.yml` runs the Go API, PostgreSQL, Redis, and Neo4j. The values in Compose are local development credentials only—do not reuse them outside a local machine.

```powershell
docker compose build backend
docker compose up -d postgres redis neo4j
docker compose run --rm --entrypoint /usr/local/bin/seed backend
docker compose up -d backend
```

The seed command generates 10,000 synthetic engineer profiles and 50,000 skill edges by default. It is safe to rerun with the same counts for those deterministic IDs; it does not clear unrelated graph data. To use a smaller fixture, add `-e ENGINEER_COUNT=1000 -e SKILLS_PER_ENGINEER=5` before `backend` in the `docker compose run` command.

Configure the frontend:

```powershell
cd frontend
npm ci
```

Create `frontend/.env.local`:

```dotenv
NEXT_PUBLIC_API_URL=http://localhost:8080
```

Then run `npm run dev`. The API listens on port `8080`, and the default allowed browser origin is `http://localhost:3000`.

## Deploying the API

Deploy the `backend` directory with Go 1.22 or build the supplied `backend/Dockerfile`. The API requires reachable PostgreSQL, Redis, and Neo4j services; it exits with an explicit startup error if a required service cannot be configured or reached.

Set these backend environment variables in the hosting provider:

| Variable | Purpose |
|---|---|
| `PORT` | HTTP port supplied by the host |
| `FRONTEND_ORIGIN` | Exact browser origin allowed by CORS, e.g. `https://your-site.example` |
| `POSTGRES_URL` | PostgreSQL connection URL |
| `REDIS_URL` | Redis connection URL |
| `NEO4J_URI` | Neo4j Bolt URI |
| `NEO4J_USERNAME` | Neo4j username |
| `NEO4J_PASSWORD` | Neo4j password |

Use managed production databases, TLS where supported, unique credentials, and a secret manager. Do not copy the Compose credentials into a hosted environment. Run the seed command against the intended Neo4j database before demoing expert matching.

For Vercel, set the project root to `frontend` and configure `NEXT_PUBLIC_API_URL` to the public API origin. The API's `FRONTEND_ORIGIN` must exactly match the deployed frontend origin.

## Latency considerations

The Redis feed cache reduces repeated `GET /api/v1/bounties` database reads; cache misses and new-request matching still use the backing services. An end-to-end 200 ms target also depends on avoiding API cold starts and keeping the API, PostgreSQL, Redis, and Neo4j in nearby regions. Measure warm and cold requests separately and report p50/p95 for a stated workload; this repository does not establish a 200 ms result.

## Validation

With Go 1.22+, run `cd backend; go test ./...`. The PostgreSQL/Redis integration test is opt-in through `TEST_POSTGRES_URL` and `TEST_REDIS_URL`; the Neo4j ranking test is opt-in through `TEST_NEO4J_URI`, `TEST_NEO4J_USERNAME`, and `TEST_NEO4J_PASSWORD`. Point these at disposable test services. Check `GET /api/v1/health` after deployment.

No latency, throughput, or matching-quality benchmark is included. Measure those under a documented workload before adding numeric performance claims to a resume.
