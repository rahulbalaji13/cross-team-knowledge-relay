# Deployment guide (Render backend + Vercel frontend)

## Backend (Render)

- **Root directory:** `backend`
- **Build command:** `go build -o server ./cmd/server/main.go`
- **Start command:** `./server`
- **Environment variables:**
  - `PORT=8080`

After deploy, verify:
- `GET https://<render-app>.onrender.com/`
- `GET https://<render-app>.onrender.com/api/v1/health`

## Frontend (Vercel)

- **Root directory:** `frontend`
- **Framework:** Next.js (auto)
- **Environment variables:**
  - `NEXT_PUBLIC_API_URL=https://<render-app>.onrender.com`

Optional (future websocket):
- `NEXT_PUBLIC_WS_URL=https://<render-app>.onrender.com`

## Why you saw the default Next.js template

Your deployed frontend rendered the default scaffold because `src/app/page.tsx` still contained the starter template. This repository now includes a production page wired to your backend API.

## Post-deploy smoke tests

```bash
curl -i https://<render-app>.onrender.com/api/v1/health
curl -i https://<render-app>.onrender.com/api/v1/bounties
curl -i -X POST https://<render-app>.onrender.com/api/v1/bounties \
  -H 'Content-Type: application/json' \
  -d '{"title":"Kafka lag issue","description":"Need debugging help","skills":["Kafka","Go"],"bounty_amount":200,"ttl_seconds":3600}'
```
