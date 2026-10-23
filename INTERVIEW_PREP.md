# Cross-Team Knowledge Relay — Technical Interview Defense Guide

## 1. Project Overview

### Purpose
Cross-Team Knowledge Relay is a monorepo demo for routing internal engineering blockers to qualified experts outside the requester's own team. A user posts a bounty with title, description, skills, poster team, reward amount, and TTL; the Go API normalizes the request, scores seeded experts, excludes same-team experts, stores the in-memory bounty, and returns ranked cross-team matches.

### Real-world problem solved
Large engineering organizations develop knowledge silos. Engineers often do not know who outside their immediate team can unblock Kafka, Neo4j, Go, frontend, or distributed-system issues. This project demonstrates a marketplace-style knowledge relay where rewards, expert ranking, graph relationships, and escrow mechanics can incentivize knowledge sharing.

### Target users
- Engineers blocked by a specialized technical issue.
- Senior/staff engineers who can help across teams.
- Engineering managers and platform teams trying to reduce duplicate debugging effort.
- Internal developer-product teams evaluating expertise discovery.

### End-to-end workflow
1. User opens the Next.js home page.
2. Frontend calls `GET /api/v1/bounties` to load current bounty feed.
3. User submits a bounty form.
4. Frontend sends `POST /api/v1/bounties` with title, description, skills, amount, TTL, and poster team.
5. Backend validates required fields, normalizes skills, computes expert matches, prepends the bounty to an in-memory slice, and responds `202 Accepted`.
6. Frontend inserts the returned bounty into UI state and displays matched experts.
7. Escrow release is represented by a stub HTTP handler in the runnable server and a more complete GORM/Redis state-machine service in `backend/internal/domain/state/escrow.go`.

### High-level architecture diagram
```text
[Browser / Next.js App Router]
  | GET /api/v1/bounties
  | POST /api/v1/bounties
  v
[Go net/http API]
  |-- CORS middleware
  |-- /health, /experts, /bounties, /escrow/*/release
  |-- in-memory bounty feed guarded by sync.RWMutex
  |-- seeded expert directory
  v
[Matching Logic]
  |-- normalize skills
  |-- exclude same-team experts
  |-- score = skill level + recency decay + reputation
  |-- sort descending and cap top 3
  v
[Frontend UI State]
  |-- metrics cards
  |-- live feed cards
  |-- create bounty form

Planned/architectural layer documented but not fully wired:
[Neo4j graph matching] [PostgreSQL escrow/transactions] [Redis idempotency/pubsub] [Socket.io realtime]
```

## 2. Tech Stack Analysis

| Layer | Current implementation | Why chosen | Alternatives/tradeoffs |
|---|---|---|---|
| Frontend | Next.js 16 App Router, React 19, TypeScript | Fast single-page dashboard, strong typing, Vercel-native deployment | Vite is simpler; Remix has stronger data mutations; plain React has less framework complexity |
| Styling | Tailwind CSS 4 via PostCSS | Utility classes keep the demo compact and responsive | CSS Modules improve encapsulation; MUI/Chakra are faster for enterprise widgets but heavier |
| Backend | Go 1.20, `net/http` | Small binary, simple concurrency, easy interview explanation | Gin/Fiber add ergonomics; Node would align with frontend; Java/Spring better enterprise conventions |
| Persistence | In-memory slice in runnable demo; planned PostgreSQL/Neo4j/Redis in docs and domain code | In-memory keeps demo deployable; graph DB fits expertise traversal; Postgres fits escrow ACID; Redis fits idempotency/pubsub | Pure Postgres with recursive queries simplifies ops; Elasticsearch can do skill search but not graph semantics as naturally |
| Realtime | `socket.io-client` hook exists; backend websocket not implemented | Shows intended push-based matched-bounty UX | Server-Sent Events simpler; raw WebSocket lighter; polling easiest but less real-time |
| Deployment | Frontend Vercel, backend Render, Docker Compose for local dependencies | Common free-tier deployment path | AWS ECS/EKS for production; Fly.io/Railway simpler monorepo deploys |

## 3. File-by-file Breakdown

| File | Purpose / why it exists | If removed | Key elements and data flow | Interview question → strong answer |
|---|---|---|---|---|
| `README.md` | Staff-level architecture narrative, graph model, escrow states, scaling plan, sample API contracts. | Project loses its conceptual design and resume framing. | Documents frontend, API gateway, Go services, Neo4j, Postgres, Redis, Kafka/Redis Streams, WebSocket notifications. | Q: Why graph + relational? A: Relationships and expert traversal belong in Neo4j; escrow and financial transitions require ACID in Postgres. |
| `DEPLOYMENT.md` | Step-by-step Render/Vercel deployment guide with env vars. | Deployers may build from wrong monorepo root. | Backend root `backend`; frontend root `frontend`; env vars include `PORT`, `NEO4J_URI`, `REDIS_URL`, `POSTGRES_URL`, `NEXT_PUBLIC_WS_URL`. | Q: Why separate hosts? A: Vercel optimizes Next.js; Render can run Go services with independent scaling. |
| `docker-compose.yml` | Local Postgres, Redis, Neo4j, backend composition. | Local infra story disappears. Also current backend build references a missing Dockerfile, so it is aspirational until added. | Defines ports 8080, 5432, 6379, 7474, 7687 and service env vars. | Q: What's broken? A: `backend/Dockerfile` is absent; add it or remove backend service from compose. |
| `vercel.json` | Root-level Vercel routing/build config targeting `frontend/package.json`. | Root deployments may not find frontend. | Uses `@vercel/next`, filesystem route, fallback to `/`. | Q: Why also `frontend/vercel.json`? A: It supports both root and frontend-root deployment modes; duplicated config should be simplified. |
| `docs/architecture/README.md` | Explains HLD and LLD SVG diagrams. | Diagram purpose is less discoverable. | Links to `hld-architecture.svg` and `lld-architecture.svg`. | Q: What do HLD/LLD show differently? A: HLD shows system boundaries; LLD shows bounty request sequence. |
| `docs/architecture/hld-architecture.svg` | High-level architecture image. | Visual system-design artifact lost. | Shows client, API edge, services, storage, async matching, notifications, fallback. | Q: Is it exact implementation? A: No; it is target architecture beyond the runnable demo. |
| `docs/architecture/lld-architecture.svg` | Low-level sequence-style architecture image. | Detailed flow artifact lost. | Walks request DTO, validation, graph write/query, async processing, ranking, notification payloads. | Q: Why sequence-style? A: It ties implementation stages to a single bounty creation flow. |
| `backend/go.mod` | Declares Go module path and Go version. | Go tooling cannot resolve module. | Module is `github.com/rahulbalaji13/cross-team-knowledge-relay/backend`; currently lacks GORM/Redis deps needed by escrow package. | Q: Why do tests fail? A: `escrow.go` imports GORM and Redis but `go.mod` does not require them. |
| `backend/cmd/server/main.go` | Runnable HTTP API and matching demo. | Backend app disappears. | Defines DTOs, seed experts, routes, CORS, validation, matching, in-memory bounty storage, escrow release stub. | Q: Why `sync.RWMutex`? A: The bounty slice is shared across requests; read/write locks prevent race conditions. |
| `backend/internal/domain/state/escrow.go` | Domain-level escrow state machine with Redis idempotency and GORM transaction locking. | Interview story for safe payouts is weaker. | `ReleaseFunds` uses Redis `SetNX`, DB transaction, `SELECT FOR UPDATE`, status guard, commit. | Q: How prevent double payout? A: Idempotency key short-circuits duplicates and row lock serializes concurrent releases. |
| `backend/internal/infrastructure/neo4j/queries.cypher` | Target Cypher query for expert ranking. | Graph-matching implementation reference lost. | Matches bounty required skills to expert `HAS_SKILL`, excludes poster team, scores by skill/reputation/recency. | Q: Why exclude same team? A: Product goal is cross-team knowledge transfer, not local escalation. |
| `backend/scripts/seed.go` | Stub for synthetic MAANG-scale dataset generation. | No documented data-generation plan. | Prints generation steps for 10K engineers, 50K skill edges, 5K bounties. | Q: Is it production-ready? A: No, it is a placeholder requiring Neo4j/Postgres client writes and deterministic seeding. |
| `frontend/package.json` | Frontend scripts and dependencies. | npm cannot install or run app. | Scripts `dev`, `build`, `start`, `lint`; deps Next, React, Socket.io client, Tailwind/TS/ESLint. | Q: Why TypeScript? A: It protects API response and state shape assumptions at compile time. |
| `frontend/package-lock.json` | Reproducible npm dependency tree. | Install versions may drift. | Locks Next/React/Tailwind/ESLint transitive dependencies. | Q: Commit lockfiles? A: Yes for applications to make CI and production builds repeatable. |
| `frontend/README.md` | Default Next.js onboarding doc. | New developers lose generic startup instructions. | Mentions `npm run dev` and editing app page, but has stale default references. | Q: What would you improve? A: Replace with project-specific frontend guide and API env docs. |
| `frontend/next-env.d.ts` | Next.js TypeScript ambient declarations. | TypeScript loses Next-specific types. | Generated file referenced by `tsconfig`. | Q: Should you edit it? A: No, Next generates it. |
| `frontend/next.config.ts` | Next.js configuration placeholder. | Defaults still work; future config location lost. | Exports empty `NextConfig`. | Q: Why keep it? A: It is the standard extension point for images, rewrites, output mode, etc. |
| `frontend/vercel.json` | Frontend-root Vercel config. | Frontend-root deploy still likely works by auto-detection; explicit version lost. | Minimal version 2 config. | Q: Root vs nested config? A: Pick one deployment mode to avoid confusion. |
| `frontend/eslint.config.mjs` | ESLint flat config for Next core web vitals and TS. | Lint command lacks project rules. | Imports `eslint-config-next` configs and ignores build outputs. | Q: Why lint? A: It catches React/Next correctness issues before deploy. |
| `frontend/tsconfig.json` | TypeScript compiler settings. | TS compilation cannot run correctly. | Strict mode, no emit, bundler resolution, React JSX, Next plugin, alias `@/*`. | Q: Why strict? A: Interview-grade code should fail early on null/shape mistakes. |
| `frontend/postcss.config.mjs` | Wires Tailwind CSS PostCSS plugin. | Tailwind import in globals may not process. | Plugin `@tailwindcss/postcss`. | Q: Why PostCSS? A: Next uses it in the CSS build pipeline. |
| `frontend/src/app/layout.tsx` | Root App Router layout and metadata. | App lacks required root layout and metadata. | Wraps children in `<html lang="en"><body>`. | Q: Why metadata here? A: App Router centralizes SEO/document metadata in layouts/pages. |
| `frontend/src/app/page.tsx` | Main client page, form, metrics, API calls. | Product UI disappears. | React state stores form and bounties; `loadBounties` GETs feed; `submitBounty` POSTs form; UI renders cards. | Q: Why client component? A: It uses hooks, local form state, effects, and browser fetch. |
| `frontend/src/app/globals.css` | Global Tailwind import, theme variables, body styles. | Styling/theme breaks. | Defines light/dark CSS variables and Inter/Arial fallback. | Q: Why CSS variables? A: They enable theme values to be consumed by Tailwind and browser media queries. |
| `frontend/src/app/favicon.ico` | Browser tab icon. | App shows default/missing favicon. | Static binary asset served by Next. | Q: Does it affect logic? A: No, only branding. |
| `frontend/hooks/useBountyStream.ts` | Unused Socket.io realtime hook for matched bounty pushes. | Realtime design artifact disappears. | Connects to `NEXT_PUBLIC_WS_URL`, listens `bounty.matched` and `bounty.expired`, dedupes state. | Q: What's the gap? A: No backend Socket.io server currently emits these events. |
| `frontend/public/*.svg` | Default public SVG assets from create-next-app. | No runtime impact unless referenced. | Static assets `file`, `globe`, `next`, `vercel`, `window`; currently unused. | Q: Should they stay? A: Remove unused scaffold assets for polish. |

## 4. Frontend Analysis

### Page: `/` (`frontend/src/app/page.tsx`)
- **Purpose:** Dashboard for creating bounties and viewing expert matches.
- **Routing:** Next.js App Router maps `src/app/page.tsx` to `/`.
- **Components used:** Inline `Home` and `MetricCard` functions; root wrapping from `layout.tsx`.
- **State management:** Local React `useState` for form fields, bounty feed, status, and connection flag; `useMemo` derives total reward value and match count.
- **API calls:** `GET ${API_URL}/api/v1/bounties`; `POST ${API_URL}/api/v1/bounties`.
- **Validation:** Browser `required`/`min` attributes plus backend validation. Skills are split by comma and empty entries filtered.
- **Error handling:** Non-OK fetch throws; status banner shows backend URL and error message.
- **Interview defense:** "I kept page state local because the app has one page and no cross-route state. If we added auth, notifications, and multiple pages, I would introduce server actions or a query cache such as TanStack Query."

### Hook: `useBountyStream`
- **Purpose:** Planned realtime feed for targeted bounty matches.
- **Routing:** Not a route; imported by future components.
- **State:** Local `bounties` and `connected`.
- **API:** Socket.io connection using engineer id query param.
- **Validation/error handling:** Reconnection attempts and timeout configured; no auth token validation yet.
- **Interview defense:** "This hook describes the intended event contract even though the current backend is REST-only. I would either implement a Socket.io-compatible backend or replace this with SSE to reduce infrastructure."

## 5. Backend Endpoint Analysis

| Endpoint | Method | Purpose | Request | Response |
|---|---:|---|---|---|
| `/` | GET/any | Service status root | none | `{"status":"online","service":"Cross-Team Knowledge Relay API"}` |
| `/api/v1/health` | GET/any | Health probe | none | `{"status":"ok"}` |
| `/api/v1/experts` | GET | Return seeded experts | none | `{"experts":[...]}` |
| `/api/v1/bounties` | GET | Return in-memory bounty feed | none | `{"bounties":[...]}` |
| `/api/v1/bounties` | POST | Create bounty and compute matches | JSON body: `title`, `description`, `skills`, `bounty_amount`, `ttl_seconds`, `poster_team` | `202` with `status`, `message`, `bounty` |
| `/api/v1/escrow/{id}/release` | POST | Stub escrow release response | Header `X-Idempotency-Key` required | `{"status":"released"}` |

### Endpoint details and likely questions
- **Root:** No method guard. Q: Why have it? A: Quick service smoke test for humans and PaaS logs.
- **Health:** No dependency checks. Q: Is it production-grade? A: No; production should check DB/Redis/Neo4j readiness separately from liveness.
- **Experts:** Method-guarded GET. Q: Security concern? A: It exposes internal expert directory; production requires auth, pagination, and privacy filtering.
- **GET bounties:** Uses read lock. Q: Why read lock? A: Multiple readers can list bounties while writes are exclusive.
- **POST bounties:** Uses POST because it creates state and triggers matching. Status `202` signals accepted/async semantics, though current matching is synchronous. Q: Why 202? A: The target design dispatches matching asynchronously; the demo returns matches inline for usability.
- **POST escrow release:** Stub only validates method, suffix, and idempotency header. Q: What is missing? A: It does not parse escrow ID, call `EscrowService`, authenticate, authorize, or persist state.

## 6. API Deep Dive

### `POST /api/v1/bounties`
- **Why exists:** Core product action: convert a blocker into an expert-matching request.
- **Why POST:** It creates a new bounty and changes server state.
- **Request flow:** CORS → JSON decode → required-field validation → default poster team → `normalizeSkills` → `computeMatches` → create response object → lock and prepend to feed → return `202`.
- **Response flow:** JSON includes the new bounty and `matchedExperts` so frontend can update without a second fetch.
- **Status codes:** `202`, `400` invalid payload/missing fields, `405` unsupported method.
- **Security concerns:** No auth, open CORS, no payload size limit, no per-user authorization, no rate limiting.
- **Performance concerns:** In-memory O(number of experts × number of skills), fine for demo; use Neo4j indexes/cache/async workers at scale.
- **Design defense:** "I used synchronous in-memory scoring to make the MVP deployable, but preserved the API shape of an async workflow. In production I would persist the bounty first, publish `BountyCreated`, and fan out matches after graph processing."

### `GET /api/v1/bounties`
- **Why exists:** Allows UI to hydrate the feed on page load.
- **Why GET:** It is read-only.
- **Status codes:** `200`, `405` for unsupported methods via default branch.
- **Security/performance:** Needs pagination, auth, tenant scoping, cache headers.

### `GET /api/v1/experts`
- **Why exists:** Debug/demo visibility into expert directory.
- **Why GET:** Read-only directory query.
- **Security:** Should be admin-only or privacy-filtered in real enterprise.

### `POST /api/v1/escrow/{id}/release`
- **Why exists:** Represents payout completion after successful help.
- **Why POST:** It triggers a state transition/action, not a full resource replacement.
- **Current status codes:** `200`, `400`, `404`, `405`.
- **Production design:** Require poster authorization, verify current escrow status, enforce idempotency, lock row, update ledger, audit event.

## 7. Database Analysis

### Implemented runtime storage
The runnable server stores bounties in a process-local `[]bountyResponse` protected by `sync.RWMutex`. This is volatile: deploy restart loses data and multiple instances diverge.

### Planned PostgreSQL schema from docs and escrow model
| Model/table | Columns | Keys/indexes/constraints | Relationships |
|---|---|---|---|
| `users` | `id`, `github_id`, `name`, `reputation_score`, `created_at` | `id` PK; `github_id` unique recommended | User posts bounties and can be expert |
| `bounties` | `id`, `poster_id`, `status`, `amount`, `expires_at`, `created_at` | `id` PK; index `poster_id`, `status`, `expires_at` recommended | Belongs to poster; has escrow |
| `escrows` | `id`, `bounty_id`, `expert_id`, `status`, `amount`, `updated_at` | `id` PK; GORM indexes on `bounty_id`, `expert_id`; `amount not null` | References bounty and expert/user |

### Planned Neo4j graph
```text
(:Engineer {id, name, reputation})-[:BELONGS_TO]->(:Team {name, domain})
(:Engineer)-[:HAS_SKILL {level, last_used}]->(:Skill {name, category})
(:Bounty {id, amount, timestamp})-[:REQUIRES {weight}]->(:Skill)
(:Engineer)-[:SOLVED {rating, date}]->(:Bounty)
(:Engineer)-[:COLLABORATED_WITH {count}]->(:Engineer)
```

### ER diagram
```text
User(id PK, github_id, name, reputation_score)
  1 ─── * Bounty(id PK, poster_id FK, status, amount, expires_at)
             1 ─── 0..1 Escrow(id PK, bounty_id FK, expert_id FK, status, amount)
User(id PK) 1 ─── * Escrow(expert_id FK)
```

### Data flow diagram
```text
Bounty form -> Go DTO -> Bounty row (planned Postgres)
                      -> Bounty/Skill nodes and REQUIRES edges (planned Neo4j)
                      -> Matching query -> Expert IDs
                      -> Notification/cache (planned Redis)
                      -> UI feed
```

### Database interview answers
- **Why not only SQL?** Graph traversals like skill proximity, collaboration, and team exclusion become more natural and faster to iterate in Neo4j.
- **Why Postgres for escrow?** Financial state transitions need ACID transactions, row locks, indexes, and auditable consistency.
- **What indexes?** `Engineer(id)`, `Skill(name)`, `Team(name)`, `Bounty(id)` in Neo4j; `bounties(status, expires_at)`, `bounties(poster_id)`, `escrows(bounty_id)`, `escrows(expert_id)` in Postgres.

## 8. Data Flow Analysis

```text
User input
  -> controlled React fields: title, description, posterTeam, skills, amount, ttlSeconds
Frontend
  -> submit handler prevents default and builds JSON payload
API request
  -> POST /api/v1/bounties with Content-Type application/json
Backend
  -> CORS headers added
  -> JSON decoded into createBountyRequest
  -> title/skills/amount/TTL validated
  -> skills trimmed/deduped by normalizeSkills
  -> experts looped; same team skipped; matching skill levels and recency decay scored
Database/storage
  -> current demo prepends to in-memory slice under write lock
Response
  -> 202 JSON includes complete bounty with matched experts
UI update
  -> setBounties prepends returned bounty, clears title/description, updates status banner
```

## 9. Authentication & Security

### Current state
- **Authentication:** None implemented. README mentions GitHub OAuth as target architecture.
- **Authorization:** None implemented. Any caller can read experts, create bounties, and release escrow stub with only an idempotency header.
- **Password handling:** None.
- **JWT/session flow:** Not implemented.
- **Input validation:** Backend validates title non-empty, skills length > 0, amount >= 1, TTL >= 60; frontend uses `required` and `min`.
- **SQL injection:** Runtime server uses no SQL. Escrow GORM query parameterizes `id = ?`, protecting against injection.
- **XSS:** React escapes rendered strings by default; however, backend should still validate/sanitize and impose length limits.
- **CSRF:** Not currently relevant without cookies/sessions, but open POST endpoints would need CSRF protection if cookie auth is added.
- **CORS:** `Access-Control-Allow-Origin: *` is acceptable for demo but too permissive for authenticated production APIs.

### Security vulnerabilities to discuss honestly
1. No authn/authz.
2. Open CORS.
3. No rate limiting.
4. No request body size limit.
5. No persistence/audit trail for bounties.
6. Escrow HTTP handler is stubbed and not wired to the real `EscrowService`.
7. Experts endpoint leaks internal directory data.
8. Socket hook passes identity as query param, not a signed token.

### Interview defense
"The demo emphasizes architecture and matching flow. For production, the first security milestone is GitHub/enterprise SSO, JWT/session validation middleware, per-tenant RBAC, body limits, structured validation, rate limiting, restrictive CORS, audit logs, and wiring escrow release to the transactional service."

## 10. Dependency Analysis

### Backend
- `encoding/json`, `net/http`, `sync`, `time`, `math`, `sort`, `strings`, `os`, `log`: standard library keeps server dependency-free and easy to deploy.
- `gorm.io/gorm`, `gorm.io/gorm/clause`: used in escrow domain for ORM and row locking, but missing from `go.mod`.
- `github.com/go-redis/redis/v8`: used in escrow idempotency, also missing from `go.mod`.

### Frontend packages
| Package | Why used | Benefits | Alternatives | Interview angle |
|---|---|---|---|---|
| `next` | App framework | Routing, build, Vercel support | Vite, Remix | Explain App Router and client component boundary |
| `react`, `react-dom` | UI primitives | Hooks, state, component model | Vue, Svelte | Explain controlled forms and rendering |
| `socket.io-client` | Realtime hook | Reconnection and event protocol | SSE, native WS | Explain why not used yet |
| `typescript` | Static types | Safer DTO/UI state | JavaScript | Explain strict mode |
| `tailwindcss`, `@tailwindcss/postcss` | Styling | Fast utility CSS | CSS Modules, MUI | Explain responsive utility classes |
| `eslint`, `eslint-config-next` | Quality checks | Next/React lint rules | Biome | Explain CI quality gate |

## 11. Deployment Analysis

- **Build process:** Backend: `go build -o server ./cmd/server/main.go`; frontend: `next build` via `npm run build`.
- **Environment variables:** Backend uses `PORT`; docs mention `NEO4J_URI`, `NEO4J_AUTH`, `REDIS_URL`, `POSTGRES_URL`. Frontend uses `NEXT_PUBLIC_API_URL` in code and `NEXT_PUBLIC_WS_URL` in realtime hook/deployment docs.
- **CI/CD:** No GitHub Actions present. Vercel/Render can auto-deploy from GitHub.
- **Hosting:** Vercel for frontend, Render for Go backend, managed Render Postgres, Upstash Redis, Neo4j Aura.
- **Production architecture:** Stateless API replicas behind load balancer, persistent Postgres/Neo4j/Redis, async worker for matching, realtime notification service.
- **Scaling:** Horizontally scale frontend/API; use Neo4j read replicas and indexes; Redis cache for hot skill-to-expert mappings; queue matching work; paginate feeds.
- **Known deployment issue:** `docker-compose.yml` points to `backend/Dockerfile`, which is absent.

## 12. Performance Analysis

- **Current bottlenecks:** In-memory storage, O(E×S) matching scan, no pagination, no cache, no persistence, process-local state.
- **Backend optimizations:** Replace scan with Neo4j indexed query; precompute expert skill vectors; async matching; add pagination; body limits; structured logs; pprof/metrics.
- **Database optimizations:** Index skill/name/team/status; cache top experts per skill; denormalize summary fields; use queue to smooth spikes.
- **Frontend optimizations:** Use TanStack Query/SWR caching; optimistic updates; loading skeletons; component extraction; avoid rendering unbounded feeds.
- **Caching:** Cache expert directory, skill normalization map, match results for common skill combinations, and health/static endpoints.

## 13. System Design Round Preparation

- **Why this architecture?** "Expert matching is graph-heavy, escrow is transactional, notifications are event-driven, and the UI needs fast feedback. Splitting those concerns lets each storage/service choice match its consistency and query pattern."
- **Scale to 1M users:** Add SSO/tenant boundaries, API gateway/rate limits, stateless Go replicas, persistent Postgres with replicas/partitioning, Neo4j cluster/read replicas, Redis cache, Kafka/Redis Streams workers, websocket fanout, observability, and disaster recovery.
- **High traffic:** Accept bounty quickly, persist it, enqueue matching, return 202, process asynchronously, push notifications. Apply backpressure and degrade to cached skill index when Neo4j is slow.
- **Current limitations:** No auth, no persistence in runnable path, missing Dockerfile, missing Go deps for escrow package, no tests, no CI, no websocket backend.
- **Improve with more time:** Wire real DBs, implement auth/RBAC, add migrations/tests, build matching worker, add Dockerfile/CI, replace scaffold docs/assets, add observability/security hardening.

## 14. Interview Defense Mode

### 100 beginner questions
1. What does the app do? → It lets engineers post blockers as bounties and receive cross-team expert matches. Follow-up: Why cross-team? → To reduce silos and spread knowledge.
2. What is the frontend framework? → Next.js with React and TypeScript. Follow-up: Why? → Fast routing/builds and type-safe UI.
3. What is the backend language? → Go. Follow-up: Why Go? → Simple concurrency and deployable static-ish binaries.
4. Where is the main page? → `frontend/src/app/page.tsx`. Follow-up: Why `app`? → Next App Router convention.
5. Where is the server entry point? → `backend/cmd/server/main.go`. Follow-up: Why `cmd/server`? → Standard Go layout for executable entrypoints.
6. What endpoint creates bounties? → `POST /api/v1/bounties`. Follow-up: Why POST? → It creates server state.
7. What endpoint lists bounties? → `GET /api/v1/bounties`. Follow-up: Why GET? → It is read-only.
8. How are skills entered? → Comma-separated string in the form. Follow-up: How normalized? → Trimmed, empty entries removed, duplicates removed case-insensitively.
9. What is a bounty TTL? → Expiration duration in seconds. Follow-up: Minimum? → Backend requires at least 60 seconds.
10. What is the reward amount? → Integer dollar-like amount. Follow-up: Minimum? → Backend requires at least 1.
11. What is `posterTeam`? → Team of the requester. Follow-up: Why needed? → To exclude same-team experts.
12. What is `MatchedExpert`? → Frontend/backend DTO for expert recommendation. Follow-up: Fields? → ID, name, team, score, matched skills.
13. How are bounties stored now? → In memory. Follow-up: Risk? → Lost on restart and not shared across replicas.
14. What protects in-memory bounties? → `sync.RWMutex`. Follow-up: Why? → Avoid concurrent read/write races.
15. What does CORS middleware do? → Adds cross-origin and method headers. Follow-up: Risk? → Wildcard origin is unsafe for authenticated production.
16. What does `/health` return? → `{"status":"ok"}`. Follow-up: Is it deep? → No, it does not check dependencies.
17. What is `useMemo` used for? → Derived totals. Follow-up: Is it required? → Not strictly, but avoids recomputing on unrelated renders.
18. What is `useEffect` used for? → Initial bounty load. Follow-up: Dependency array? → Empty, so it runs once on mount.
19. Why is page a client component? → It uses hooks and browser events. Follow-up: How marked? → `"use client"` directive.
20. What is Tailwind? → Utility CSS framework. Follow-up: Where imported? → `globals.css`.
21. What is `layout.tsx`? → Root layout and metadata. Follow-up: Why needed? → App Router requires root layout.
22. What is `package-lock.json`? → Locks npm versions. Follow-up: Why commit? → Reproducible installs.
23. What is `go.mod`? → Go module descriptor. Follow-up: Current issue? → Missing GORM/Redis deps used by escrow package.
24. What does `normalizeSkills` return? → Trimmed deduped skills. Follow-up: Preserves casing? → Yes, display casing of first occurrence.
25. What does `decay` do? → Computes exponential recency score. Follow-up: Missing date? → Defaults to 0.5.
26. What is `math.Exp(-0.03*days)`? → Time decay formula. Follow-up: Why decay? → Recent skill use is more valuable.
27. Why sort matches? → Highest score first. Follow-up: How many returned? → Top 3 in runnable server.
28. What is the experts data source? → Seeded slice in Go. Follow-up: Production? → Neo4j/Postgres-backed directory.
29. What is Neo4j for? → Graph relationships between engineers, skills, teams, bounties. Follow-up: Why graph? → Natural relationship traversal.
30. What is Postgres for? → Transactional bounty/escrow data. Follow-up: Why? → ACID guarantees.
31. What is Redis for? → Idempotency, rate limiting, pub/sub/cache in target design. Follow-up: Current usage? → Escrow domain code imports Redis, not wired in server.
32. What is Socket.io for? → Realtime bounty notifications. Follow-up: Current gap? → No backend Socket.io server.
33. What does `DEPLOYMENT.md` explain? → Render backend and Vercel frontend deployment. Follow-up: Key setting? → Correct root directory.
34. What does root `vercel.json` do? → Builds frontend from monorepo root. Follow-up: Possible issue? → Duplicate Vercel config can confuse deploys.
35. What does Docker Compose start? → Backend, Postgres, Redis, Neo4j. Follow-up: Broken part? → Missing backend Dockerfile.
36. What is `queries.cypher`? → Planned Neo4j matching query. Follow-up: Same-team exclusion? → `WHERE NOT expert belongs to posterTeam`.
37. What is escrow? → Held funds for bounty payout. Follow-up: States? → CREATED, HELD, MEETING_CONFIRMED, RELEASED, REFUNDED.
38. What is idempotency? → Same request can be safely retried. Follow-up: How implemented? → Redis `SetNX` key.
39. What is `SELECT FOR UPDATE`? → Row lock during transaction. Follow-up: Why? → Prevent concurrent state changes.
40. What status does POST bounty return? → 202 Accepted. Follow-up: Why not 201? → Target matching is asynchronous.
41. What status for bad payload? → 400. Follow-up: Where? → JSON decode/validation branch.
42. What status for wrong method? → 405. Follow-up: Which endpoints? → Experts, bounties, escrow release.
43. What is `Content-Type` set to? → `application/json`. Follow-up: In middleware? → Yes.
44. What does frontend default API URL do? → Points to deployed Render backend. Follow-up: Override? → `NEXT_PUBLIC_API_URL`.
45. What is `NEXT_PUBLIC_*`? → Browser-exposed env variable. Follow-up: Secret? → Never put secrets there.
46. What is `cache: no-store`? → Avoids cached GET response. Follow-up: Why? → Feed should be fresh.
47. What happens after successful submit? → UI prepends returned bounty and clears title/description. Follow-up: Does it clear skills? → No, defaults remain.
48. What happens on error? → Status banner shows failure. Follow-up: Better UX? → Field errors/toasts/retry.
49. Does frontend validate skill duplicates? → It filters empty skills only. Follow-up: Where dedupe? → Backend.
50. Does backend validate title length? → No. Follow-up: Risk? → Abuse/memory/UI issues.
51. Does backend authenticate release? → No. Follow-up: Production fix? → Auth middleware and ownership check.
52. Is `EscrowService` used by HTTP handler? → No. Follow-up: Why mention? → It models intended domain logic.
53. What is GORM? → Go ORM. Follow-up: Why use? → Transactions and model mapping.
54. What is a primary key in Escrow? → `ID`. Follow-up: GORM tag? → `gorm:"primaryKey"`.
55. Which Escrow fields are indexed? → `BountyID`, `ExpertID`. Follow-up: Why? → Common lookup paths.
56. What does `Amount not null` mean? → DB should reject null amount. Follow-up: Is int nullable in Go? → No, zero default still possible.
57. What is `UpdatedAt`? → Timestamp for modifications. Follow-up: GORM behavior? → Auto-managed if configured conventionally.
58. What is `seed.go`? → Stub data generator. Follow-up: Does it insert records? → No, only prints.
59. What do SVG diagrams provide? → Architecture visuals. Follow-up: Are they generated? → Static checked-in SVGs.
60. What is an HLD? → High-level design. Follow-up: LLD? → Low-level implementation sequence.
61. What is a monorepo? → Frontend and backend in one repo. Follow-up: Benefit? → Shared versioning and easier demo deploy.
62. What is a tradeoff of monorepo? → Tooling/deploy root complexity. Follow-up: Evidence? → Deployment docs emphasize root directories.
63. What is a controlled input? → Input value driven by React state. Follow-up: Benefit? → Easy validation and submission.
64. Why use `FormEvent`? → Type-safe submit handler. Follow-up: From where? → React.
65. What is `preventDefault`? → Prevents browser form navigation. Follow-up: Why? → SPA handles submit via fetch.
66. Why use `Number(e.target.value)`? → Convert input string to number. Follow-up: Risk? → Empty string becomes 0.
67. Why response field `expiresAt` camelCase? → Frontend-friendly JSON. Follow-up: Request uses snake_case? → Backend DTO accepts snake_case for API contract.
68. What is inconsistent naming risk? → Mapping bugs. Follow-up: Fix? → Shared OpenAPI/schema types.
69. What is `any` in Go JSON map? → Alias for interface{}. Follow-up: Go version? → Requires Go 1.18+; module uses 1.20.
70. Why no database migrations? → Demo scope. Follow-up: Production? → Use Goose/Flyway/Atlas.
71. Why no tests? → Current weakness. Follow-up: Add first? → Unit tests for normalize/decay/computeMatches and handler tests.
72. What is rate limiting? → Restrict request volume. Follow-up: Where add? → Middleware with Redis.
73. What is pagination? → Limit/offset or cursor. Follow-up: Needed where? → Bounty/expert feeds.
74. What is observability? → Logs, metrics, traces. Follow-up: Current? → Basic log only.
75. What is a circuit breaker? → Stops calls to unhealthy dependencies. Follow-up: Mentioned where? → README scaling strategy.
76. What is backpressure? → Handling overload gracefully. Follow-up: Example? → Drop noncritical websocket updates.
77. What is `StatusAccepted`? → HTTP 202. Follow-up: Semantics? → Request accepted for processing.
78. What is `StatusNoContent`? → HTTP 204. Follow-up: Used for? → CORS preflight OPTIONS.
79. Why OPTIONS? → Browser CORS preflight. Follow-up: What headers allowed? → Origin, Content-Type, Accept, X-Idempotency-Key.
80. What is `strings.HasSuffix` used for? → Check escrow release path. Follow-up: Limitation? → Does not extract/validate ID.
81. What is `strings.EqualFold`? → Case-insensitive comparison. Follow-up: Used where? → Team exclusion.
82. What is `strings.TrimSpace`? → Removes surrounding whitespace. Follow-up: Used where? → Title/team/skills validation.
83. What is `sort.Slice`? → Custom sort. Follow-up: Comparator? → Higher score first.
84. What is a memory leak risk? → Unbounded bounty slice. Follow-up: Fix? → DB + pagination/TTL cleanup.
85. What is `favicon.ico`? → Browser icon. Follow-up: Logic impact? → None.
86. What is `next-env.d.ts`? → Next TypeScript declarations. Follow-up: Edit manually? → No.
87. What is `tsconfig` strict? → Strong type checking. Follow-up: Benefit? → Fewer runtime type bugs.
88. What is ESLint core web vitals? → Next lint rules for best practices. Follow-up: Example? → Image/link/performance rules.
89. What is `private: true`? → Prevents accidental npm publish. Follow-up: Why good? → App package not a library.
90. What is `start` script? → Runs production Next server. Follow-up: Build first? → Yes.
91. What is `lint` script? → Runs ESLint. Follow-up: Current result? → Passes.
92. What is `go fmt`? → Go formatter. Follow-up: Did it change? → It would normalize import order/whitespace if needed.
93. What is a DTO? → Data transfer object. Follow-up: Example? → `createBountyRequest`.
94. What is a domain model? → Business entity/state. Follow-up: Example? → `Escrow`.
95. What is infrastructure code? → DB-specific implementation. Follow-up: Example? → `queries.cypher`.
96. What is product metric shown? → Open bounties, total value, matches, backend URL. Follow-up: Derived from? → Bounties state.
97. What is default poster team? → Commerce in frontend, Unknown in backend if blank. Follow-up: Why both? → UX default and backend resilience.
98. What is a sample expert? → Sara Ali, Platform, Go/Redis/distributed systems. Follow-up: Why seeded? → Demo repeatability.
99. What is current license? → None present. Follow-up: Risk? → Ambiguous reuse rights.
100. What should you say if asked whether all README features are implemented? → Be honest: some are target architecture, the runnable demo implements REST matching only. Follow-up: Why still valuable? → It shows roadmap and design thinking.

### 50 intermediate questions
1. Explain scoring. → For each matching skill, add 60% skill level normalized plus 40% recency decay, then add reputation/5; sort descending. Follow-up: Bias? → More skills accumulate more score.
2. Why same-team exclusion before scoring? → Saves work and enforces product rule. Follow-up: Alternative? → Penalize instead of exclude.
3. How would you test matching? → Table-driven tests for skills/team/date/ordering. Follow-up: Time issue? → Inject clock to make decay deterministic.
4. Why is current decay non-deterministic? → Uses `time.Since`. Follow-up: Fix? → Pass `now` as dependency.
5. What happens after March/June 2026 with seeded dates? → Scores decay lower over time. Follow-up: Risk? → Demo outputs change by date.
6. How would you persist bounties? → Postgres table and repository layer. Follow-up: Why not Neo4j only? → Need transactional lifecycle and feeds.
7. How wire Neo4j? → Create graph repository using official driver, parameterized queries, indexes. Follow-up: Avoid injection? → Parameters, not string concatenation.
8. How add auth? → SSO/OAuth callback, session/JWT middleware, user context. Follow-up: Where check poster? → Handler/service authorization layer.
9. How authorize escrow release? → Only bounty poster/admin after meeting confirmation. Follow-up: Expert? → Expert should not self-release funds.
10. How prevent duplicate bounties? → Idempotency key or client-generated request ID. Follow-up: Store where? → DB unique index/Redis pending key.
11. Why `202` but matching synchronous? → API mimics target async design. Follow-up: Better REST purity? → Use 201 for MVP or actually enqueue work.
12. How handle partial failures in async matching? → Persist bounty, retry job, dead-letter queue, user-visible pending status. Follow-up: Notification failure? → Store notification for retry.
13. Why use Redis `SetNX`? → Atomic only-if-not-exists. Follow-up: Race? → Redis command itself is atomic.
14. What's wrong with returning duplicate error when Redis errors? → It conflates outage with duplicate. Follow-up: Fix? → Separate infrastructure error from duplicate false.
15. Why defer rollback after commit? → Rollback after commit is no-op in many DBs but messy. Follow-up: Better? → Conditional rollback or ignore documented no-op.
16. How improve handler errors? → Consistent JSON helper, status mapping, structured error types. Follow-up: Current issue? → `http.Error` sets text/plain unless header set earlier.
17. Is Content-Type always correct? → Middleware sets JSON before handler. Follow-up: `http.Error`? → Body is JSON-ish but helper better.
18. How limit request size? → `http.MaxBytesReader`. Follow-up: Why? → Prevent memory exhaustion.
19. How validate payload deeply? → Use validator library or manual length/range checks. Follow-up: Skills? → Max count, allowed chars, normalized canonical IDs.
20. How add pagination? → Cursor by created_at/id. Follow-up: Why cursor? → Stable under inserts.
21. How cache experts? → Redis key by skill/team filters. Follow-up: Invalidate? → On skill/profile updates.
22. How support multiple backend instances? → Replace memory with DB/cache. Follow-up: Why? → Memory state would diverge.
23. How deploy backend Docker? → Add Dockerfile multi-stage Go build. Follow-up: Compose issue? → Current compose references missing Dockerfile.
24. How run CI? → npm ci/lint/build, gofmt/go test, Docker build. Follow-up: Secret handling? → CI secrets only for deploy, not PR tests.
25. What does `useBountyStream` dedupe do? → Removes old ID then prepends new. Follow-up: Why? → Avoid duplicate event deliveries.
26. Socket identity risk? → Query param can be spoofed. Follow-up: Fix? → Signed JWT and server authorization.
27. How avoid XSS? → React escapes; avoid dangerous HTML; sanitize if rich text. Follow-up: Stored XSS? → Backend validation and output encoding.
28. How avoid CSRF? → SameSite cookies plus CSRF token or bearer token. Follow-up: Current? → No cookies/auth.
29. How implement GitHub OAuth? → OAuth app, callback, exchange code, create session. Follow-up: Enterprise? → SAML/OIDC provider preferred.
30. How model skills? → Canonical skill table/node with aliases. Follow-up: Why aliases? → "JS" vs "JavaScript".
31. How rank experts fairly? → Blend skill, recency, reputation, availability, load, diversity. Follow-up: Abuse? → Reputation fraud detection.
32. How measure success? → Time-to-unblock, match acceptance, resolution rate, repeat collaborations. Follow-up: Product risk? → Incentives may create bounty gaming.
33. How handle expired bounties? → Background job updates status/refunds. Follow-up: UI? → Hide or mark expired.
34. How handle clock skew? → Server authoritative time. Follow-up: Distributed? → NTP and DB timestamps.
35. How store money? → Integer cents/decimal, currency, ledger. Follow-up: Current amount int? → Demo simplification.
36. How audit escrow? → Append-only ledger/events. Follow-up: Why? → Compliance and dispute resolution.
37. How implement refunds? → State transition with row lock and ledger reversal. Follow-up: Idempotency? → Unique operation key.
38. How scale Neo4j reads? → Read replicas and indexed labels. Follow-up: Writes? → Single leader/batched writes.
39. How fallback if Neo4j down? → Redis inverted index skill→experts. Follow-up: Tradeoff? → Less accurate but available.
40. How monitor? → Prometheus metrics, structured logs, traces. Follow-up: Key metrics? → API latency, match latency, queue lag, DB errors.
41. How handle PII? → Minimal expert fields, access controls, retention. Follow-up: Experts endpoint? → Restrict/remove.
42. How improve frontend architecture? → Componentize form/feed/cards, API client module, query cache. Follow-up: Why? → Testability and reuse.
43. How handle env mismatch? → Standardize `NEXT_PUBLIC_API_URL` vs `NEXT_PUBLIC_WS_URL`. Follow-up: Docs currently? → Deployment doc mentions WS URL for frontend.
44. How type API responses? → Shared schema/OpenAPI or zod. Follow-up: Runtime validation? → Zod parse before state update.
45. What is optimistic UI? → Update before server confirms. Follow-up: Use here? → Could, but matching requires server response.
46. How avoid duplicate IDs? → UUID/ULID from server. Follow-up: Current ID? → Timestamp string; collision possible under same nanosecond/process weirdness.
47. How make IDs sortable? → ULID or timestamp column. Follow-up: Current benefit? → Timestamp-like ID roughly sortable.
48. How handle international teams? → Store stable team IDs, not names. Follow-up: Current risk? → Case/name changes.
49. How handle accessibility? → Labels exist; improve contrast/focus/aria status. Follow-up: Footer issue? → White text may have poor contrast in light mode.
50. How distinguish demo vs target architecture? → Be explicit: runnable MVP vs documented production design. Follow-up: Why okay? → Interviews reward honest scope and roadmap.

### 50 advanced questions
1. Design the async matching pipeline. → Persist bounty, publish event, worker queries graph, stores matches, pushes notification. Follow-up: Exactly-once? → Use idempotent consumers and unique constraints; don't claim true exactly-once across systems.
2. How would you do personalized PageRank? → Project subgraph around skills/collaboration, seed with required skills/poster context, compute weighted centrality offline/nearline. Follow-up: Online? → Use precomputed centrality plus online filters.
3. Graph vs vector search? → Graph captures explicit org relationships; vectors capture semantic issue similarity. Follow-up: Hybrid? → Use embeddings to map text to skills, then graph rank experts.
4. How handle multi-skill matching mathematically? → Weighted aggregation, minimum coverage thresholds, diversity constraints. Follow-up: Avoid one expert dominating? → Team/load caps and fairness re-ranking.
5. How avoid reinforcing popularity bias? → Exploration, decay, load balancing, calibrate reputation, cold-start boosts. Follow-up: Metric? → Distribution of matches and resolution quality.
6. How ensure financial consistency? → Ledger, DB transactions, row locks, idempotency keys, reconciliation jobs. Follow-up: External payments? → Webhook idempotency and state-machine mapping.
7. What isolation level is needed? → Row locks under read committed can work for single-row transitions; serializable for complex invariants. Follow-up: Deadlocks? → Consistent lock ordering and retries.
8. What if Redis idempotency succeeds but DB commit fails? → Retry with same key may be blocked. Follow-up: Fix? → Store operation result/status in DB idempotency table within transaction or clean key on failure carefully.
9. Why Redis idempotency TTL 24h? → Limits memory and retry window. Follow-up: Financial ops? → Prefer durable idempotency table longer than 24h.
10. How design RBAC? → Roles poster/expert/admin, resource ownership, tenant/team boundaries, policy middleware. Follow-up: ABAC? → Include team, bounty status, relationship, org.
11. How handle tenant isolation? → Tenant ID in every row/node and auth claims; composite indexes; query filters. Follow-up: Neo4j? → Label/property tenant scoping and mandatory query predicates.
12. How design migrations? → Versioned SQL/Cypher migrations, backward-compatible deploys. Follow-up: Zero downtime? → Expand-migrate-contract.
13. How handle schema drift between frontend and backend? → OpenAPI generation or shared schema package. Follow-up: Runtime? → Validate responses.
14. How load test? → k6/Vegeta scenarios for create/list/matching. Follow-up: Metrics? → p95/p99 latency, error rate, CPU, DB query time.
15. How capacity plan Neo4j? → Estimate nodes/edges, query fanout, memory page cache, indexes, replicas. Follow-up: 1M engineers? → Tens of millions of edges; precompute and cache.
16. How shard? → By org/tenant or skill domain; avoid cross-shard traversals. Follow-up: Tradeoff? → Harder global expert discovery.
17. How secure webhooks? → HMAC signatures, timestamp tolerance, idempotency, replay protection. Follow-up: Calendly? → Verify provider signature if available.
18. How build notification reliability? → Store notification rows, ack states, retry, DLQ. Follow-up: WebSocket offline? → Send email/in-app unread queue.
19. How design audit logging? → Append-only event table with actor, action, resource, before/after, request ID. Follow-up: Privacy? → Redact sensitive text.
20. How implement request tracing? → Propagate correlation ID through API, queue, worker, DB logs. Follow-up: User-facing? → Include support trace ID.
21. How handle secret rotation? → Managed secret store and rolling deploys. Follow-up: Frontend secrets? → None in `NEXT_PUBLIC`.
22. How prevent bounty spam? → Rate limit, reputation requirements, moderation, cost/payment hold. Follow-up: Abuse by experts? → Review/rating/dispute system.
23. How model disputes? → Escrow states `DISPUTED`, admin review, evidence, deadlines. Follow-up: Current states? → No disputed state yet.
24. How avoid stale expert skills? → Recency decay, profile updates, auto-import from commits/tickets, verification. Follow-up: Privacy? → Consent and minimization.
25. How handle skill taxonomy changes? → Alias table, canonical IDs, migration jobs. Follow-up: Backcompat? → Keep aliases and redirect.
26. How make matching explainable? → Return matched skills and score components. Follow-up: Current? → Only final score and matched skills.
27. How handle ranking experiments? → Feature flags, A/B buckets, offline evaluation. Follow-up: Guardrails? → Resolution rate and fairness metrics.
28. How prevent data races in Go? → Mutexes, immutable snapshots, race detector. Follow-up: Current? → RWMutex around bounties.
29. Is experts slice safe? → Read-only after init, so yes. Follow-up: If mutable? → Protect with lock or DB.
30. How handle graceful shutdown? → `http.Server` with context, signal handling, drain. Follow-up: Current? → Uses `ListenAndServe` directly, no graceful shutdown.
31. How set timeouts? → Server read/write/idle timeouts and client timeouts. Follow-up: Current? → None, vulnerable to slowloris.
32. How harden CORS? → Allowlist origins, credentials policy, env-based config. Follow-up: Current? → Wildcard.
33. How design API versioning? → `/api/v1`, backwards-compatible fields, deprecation. Follow-up: Current? → Routes already under v1.
34. How ensure response compatibility? → Add fields only, avoid renames, contract tests. Follow-up: Current issue? → Mixed snake/camel casing.
35. How structure Go packages? → `cmd`, `internal/domain`, `internal/infrastructure`, `internal/service`. Follow-up: Current? → Some planned packages absent but layout started.
36. How avoid cyclic dependencies? → Domain pure, infrastructure implements interfaces, handlers depend on services. Follow-up: Current escrow? → Domain imports GORM/Redis, so not pure; could improve.
37. Should domain import GORM/Redis? → For clean architecture, no; use repository/idempotency interfaces. Follow-up: Why current? → Simpler demo of transaction logic.
38. How handle dependency injection? → Constructors and interfaces; wire in main. Follow-up: Current? → No DI except `NewEscrowService`.
39. How test HTTP handlers? → `httptest` with requests and response recorder. Follow-up: Race tests? → `go test -race` after dependencies fixed.
40. How handle frontend e2e? → Playwright: load page, mock API, submit form. Follow-up: CI? → Run after build.
41. How secure dependency supply chain? → lockfiles, Dependabot, npm audit/govulncheck, pin versions. Follow-up: Current? → Lockfile present; no CI scanning.
42. How handle logging PII? → Structured fields and redaction. Follow-up: Current? → Only startup log.
43. How build Docker image? → Multi-stage Go build from `golang` to distroless/alpine. Follow-up: Include CA certs? → Yes if outbound TLS.
44. How deploy zero downtime? → Health checks, rolling deploy, readiness before traffic. Follow-up: In-memory problem? → Rolling deploy loses local feed.
45. How backfill graph from SQL? → Event outbox or batch ETL with checkpoints. Follow-up: Consistency? → Source of truth and reconciliation.
46. How implement outbox pattern? → Write event row in same DB transaction; relay publishes. Follow-up: Why? → Avoid lost events between DB commit and broker publish.
47. How handle eventual consistency UX? → Show pending matching status and update when matches ready. Follow-up: Current? → Immediate matches returned.
48. How calculate SLO? → e.g., 99.9% create API availability, p95 match under 5s. Follow-up: Error budget? → Guides release/risk.
49. What is the hardest production risk? → Correct escrow and auth, because money/security bugs are high impact. Follow-up: Mitigation? → Formal state machine tests, audits, least privilege.
50. What would you change first before launch? → Implement auth, persistence, tests, and wire escrow service; remove misleading/stale scaffold. Follow-up: Why? → Trust and correctness before scaling.

## 15. Vibe-coding Risk Detection

| Area that may look AI-generated or scaffolded | What it does | Why/design/tradeoff | Questions that expose weak understanding |
|---|---|---|---|
| README describes microservices/Kafka/GitHub OAuth but code is monolithic REST | Provides target architecture beyond MVP | Good for system design, risky if claimed as implemented | "Which microservices exist in code?" Strong answer: "Only a monolithic Go demo exists; microservices are planned architecture." |
| `frontend/README.md` is default create-next-app | Generic frontend instructions | Fast scaffold, but unpolished | "Why does it mention Geist/font if layout doesn't use next/font?" Answer: "Stale scaffold; should be replaced." |
| Public SVG assets unused | Default assets | Harmless but clutter | "Where is `globe.svg` rendered?" Answer: "Nowhere." |
| `docker-compose.yml` references missing Dockerfile | Intended local infra | Shows planned containers, currently broken | "Can compose build today?" Answer: "No, not until backend Dockerfile is added." |
| Escrow service imports missing deps and is not wired | Demonstrates state-machine thinking | Useful design artifact, not runnable in current module | "Does `/escrow/.../release` call Redis/GORM?" Answer: "No, handler is stubbed." |
| Socket.io hook without server | Intended realtime client | Good future contract, incomplete implementation | "What emits `bounty.matched`?" Answer: "Nothing yet; need websocket service." |
| `202 Accepted` while doing sync matching | Aligns with async target | Semantically arguable for MVP | "Is matching async now?" Answer: "No; response code anticipates production design." |
| Seed script only prints | Placeholder scale story | Needs real data writes | "Does it generate 10K records?" Answer: "No, it is a stub." |

## 16. Resume Defense

### 30 seconds
"I built Cross-Team Knowledge Relay, a full-stack demo for helping engineers find experts outside their team. The Next.js UI lets a user post a technical bounty, and the Go API validates it, normalizes skills, scores seeded experts by skill level, recency, and reputation, excludes the poster's team, and returns ranked matches. The repo also includes a production architecture for Neo4j graph matching, Postgres escrow, Redis idempotency, and realtime notifications."

### 1 minute
"Cross-Team Knowledge Relay solves internal knowledge silos. The frontend is a Next.js/React dashboard with a bounty form and live match feed. The backend is a Go HTTP API exposing health, experts, bounty list/create, and escrow-release endpoints. The MVP stores bounties in memory with a mutex and matches against seeded experts; the target architecture separates graph matching in Neo4j, transactional escrow in Postgres, Redis for idempotency/cache/pubsub, and event-driven notification fanout. I can defend both the runnable MVP tradeoffs and the production scaling path."

### 2 minutes
"The core workflow starts when an engineer posts a blocker with skills, amount, TTL, and team. The frontend sends a POST request. The Go server decodes and validates the payload, trims and deduplicates skills, excludes experts from the same team, scores candidates using normalized skill proficiency, exponential recency decay, and reputation, then returns the top matches and stores the bounty in an in-memory feed. For production, I would persist the bounty in Postgres, create graph relationships in Neo4j, publish a `BountyCreated` event, process matches asynchronously, and push notifications over websocket/SSE. The escrow domain model shows how I would handle money safely: strict states, Redis idempotency, row-level locks, and transactional updates. The biggest honest gaps are auth, persistence, CI/tests, Dockerfile, and wiring the escrow/realtime pieces."

### 5 minutes
Use the 2-minute version, then add: "Architecturally, I chose polyglot persistence because the problem has two different data shapes: expertise discovery is graph-shaped, while financial escrow is transactional. The current Go server is intentionally simple so it can be deployed and demoed quickly, but the docs and Cypher file show the system-design direction: Neo4j for traversing Engineer-Skill-Team-Bounty relationships, Postgres for bounty/escrow state, Redis for idempotency, rate limiting, cache, and pub/sub, and Vercel/Render for deployment. If challenged, I would be transparent that some README elements are target architecture rather than implemented code. My next engineering priorities are to add auth/RBAC, replace in-memory storage with migrations and repositories, implement tests and CI, add a backend Dockerfile, and either implement Socket.io on the server or replace the client hook with SSE."

## 17. Final Interview Cheat Sheet

### Architecture summary
- Monorepo: `frontend` Next.js app + `backend` Go API + docs/deployment.
- Runnable path: REST API + in-memory feed + seeded expert matching.
- Target path: Neo4j matching, Postgres transactions, Redis idempotency/pubsub/cache, async workers, realtime notifications.

### APIs summary
- `GET /` service status.
- `GET /api/v1/health` health probe.
- `GET /api/v1/experts` seeded expert directory.
- `GET /api/v1/bounties` list current in-memory bounties.
- `POST /api/v1/bounties` validate and create bounty, compute matches.
- `POST /api/v1/escrow/{id}/release` stub release requiring idempotency header.

### Database summary
- Current: no real database in runtime; in-memory slice.
- Planned SQL: users, bounties, escrows.
- Planned graph: engineers, skills, teams, bounties and relationships.
- Escrow model has primary key, indexed bounty/expert IDs, status, amount, updated timestamp.

### Security summary
- Current: demo-level only.
- Must add: SSO, JWT/session middleware, RBAC, restrictive CORS, rate limits, request body limits, payload validation, audit logs, private expert directory, real escrow authorization.

### Deployment summary
- Frontend: Vercel with `frontend` root.
- Backend: Render with `backend` root and `go build -o server ./cmd/server/main.go`.
- Local infra intended: Compose Postgres/Redis/Neo4j, but backend Dockerfile is missing.

### Top 50 likely questions
1. What problem does this solve?
2. Why cross-team matching?
3. Why Go for backend?
4. Why Next.js for frontend?
5. Walk through POST bounty.
6. Explain `computeMatches`.
7. Explain recency decay.
8. Why exclude same-team experts?
9. Why 202 Accepted?
10. How are bounties stored?
11. What breaks with multiple backend replicas?
12. Why use a mutex?
13. What security is missing?
14. How would you add auth?
15. How would you persist data?
16. Why Neo4j?
17. Why Postgres?
18. Why Redis?
19. What is escrow?
20. How does idempotency work?
21. What does `SELECT FOR UPDATE` do?
22. Is escrow wired into the handler?
23. What endpoints exist?
24. What validation exists?
25. What validation is missing?
26. How handle XSS?
27. How handle CSRF?
28. How handle SQL injection?
29. How deploy frontend?
30. How deploy backend?
31. What env vars exist?
32. What is wrong with Docker Compose?
33. What is wrong with Go dependencies?
34. How would you test this?
35. How would you add CI?
36. How scale to 1M users?
37. How handle high traffic?
38. How handle Neo4j failure?
39. How cache matches?
40. How paginate feeds?
41. How improve frontend architecture?
42. What is `useBountyStream`?
43. Why isn't realtime working yet?
44. How would you implement notifications?
45. What files are scaffold leftovers?
46. What parts are target architecture only?
47. What would you improve first?
48. What are the hardest risks?
49. How would you explain this on resume?
50. What did you learn?

### Top 20 hardest questions
1. How do you guarantee escrow idempotency if Redis succeeds and DB fails?
2. How would you implement durable idempotency instead of Redis-only keys?
3. How would you compute PageRank at scale without blocking requests?
4. How do you prevent popularity bias in expert recommendations?
5. How do you reconcile Neo4j and Postgres consistency?
6. Would you use an outbox pattern? Why?
7. How do you do tenant isolation in Neo4j?
8. How would you design a dispute state in escrow?
9. What isolation level is sufficient for escrow release?
10. How would you migrate from in-memory storage to Postgres without downtime?
11. How would you make matching explainable to users?
12. How would you handle offline experts and availability?
13. How do you verify webhook authenticity?
14. How do you prevent bounty spam and expert gaming?
15. How would you load test graph queries?
16. How would you structure Go packages for clean architecture?
17. Why is domain importing GORM/Redis a clean-architecture smell?
18. How do you recover from queue worker poison messages?
19. How do you set SLOs for match latency and payout correctness?
20. Which README claims are not implemented, and how do you honestly present them?

### Weak areas to study before interview
- Be able to clearly separate **implemented MVP** from **planned production architecture**.
- Review Go HTTP handlers, mutexes, table-driven tests, and graceful shutdown/timeouts.
- Review OAuth/JWT/RBAC and CORS/CSRF/rate-limiting basics.
- Review Postgres transactions, row locks, idempotency tables, and ledger design.
- Review Neo4j modeling, indexes, Cypher parameterization, and graph ranking.
- Review Next.js App Router client/server component boundaries.
- Prepare an honest answer for missing Dockerfile, missing Go dependencies, absent tests/CI, unused assets, and default frontend README.
