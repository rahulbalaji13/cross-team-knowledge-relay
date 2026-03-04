# Deployment guide (Render backend + Vercel frontend)

## Production URLs

- Frontend: `https://cross-team-knowledge-relay.vercel.app`
- Backend: `https://cross-team-knowledge-relay.onrender.com`

## Backend (Render)

- **Root directory:** `backend`
- **Build command:** `go build -o server ./cmd/server/main.go`
- **Start command:** `./server`
- **Environment variables:**
  - `PORT=8080`

After deploy, verify:
- `GET https://cross-team-knowledge-relay.onrender.com/`
- `GET https://cross-team-knowledge-relay.onrender.com/api/v1/health`

## Frontend (Vercel)

- **Root directory:** `frontend`
- **Framework:** Next.js (auto)
- **Environment variables:**
  - `NEXT_PUBLIC_API_URL=https://cross-team-knowledge-relay.onrender.com`

> The frontend now also defaults to this backend URL in code, so it works even if the env var is forgotten.

## Why you saw the default Next.js template

Your deployed frontend rendered the default scaffold because `src/app/page.tsx` previously contained the starter template. This repository now includes a production page wired to your backend API.

## Post-deploy smoke tests

```bash
curl -i https://cross-team-knowledge-relay.onrender.com/api/v1/health
curl -i https://cross-team-knowledge-relay.onrender.com/api/v1/bounties
curl -i -X POST https://cross-team-knowledge-relay.onrender.com/api/v1/bounties \
  -H 'Content-Type: application/json' \
  -d '{"title":"Kafka lag issue","description":"Need debugging help","skills":["Kafka","Go"],"bounty_amount":200,"ttl_seconds":3600}'
```
