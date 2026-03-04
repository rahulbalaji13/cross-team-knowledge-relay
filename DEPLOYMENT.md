# Deployment guide (Render backend + Vercel frontend)

## Production URLs

- Frontend: `https://cross-team-knowledge-relay.vercel.app`
- Backend: `https://cross-team-knowledge-relay.onrender.com`

## What is now implemented

- Create a bounty from frontend.
- Backend computes **top 3 cross-team matches** (excluding poster team) using skill overlap + recency + reputation scoring.
- Frontend displays matched experts directly under each bounty card.

## Backend (Render)

- **Root directory:** `backend`
- **Build command:** `go build -o server ./cmd/server/main.go`
- **Start command:** `./server`
- **Environment variables:**
  - `PORT=8080`

After deploy, verify:
- `GET https://cross-team-knowledge-relay.onrender.com/`
- `GET https://cross-team-knowledge-relay.onrender.com/api/v1/health`
- `GET https://cross-team-knowledge-relay.onrender.com/api/v1/experts`

## Frontend (Vercel)

- **Root directory:** `frontend`
- **Framework:** Next.js (auto)
- **Environment variables:**
  - `NEXT_PUBLIC_API_URL=https://cross-team-knowledge-relay.onrender.com`

## Post-deploy smoke tests

```bash
curl -i https://cross-team-knowledge-relay.onrender.com/api/v1/health
curl -i https://cross-team-knowledge-relay.onrender.com/api/v1/experts
curl -i -X POST https://cross-team-knowledge-relay.onrender.com/api/v1/bounties \
  -H 'Content-Type: application/json' \
  -d '{"title":"Need Kafka + Go help","description":"Production lag","poster_team":"Commerce","skills":["Kafka","Go","Distributed Systems"],"bounty_amount":250,"ttl_seconds":3600}'
```
