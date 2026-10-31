# Knowledge Relay frontend

This Next.js app provides the bounty form, persisted request feed, and a small guided-help panel. It talks to the Go API; expert matching and persistence happen in the backend.

## Local development

From the repository root, follow the full-stack steps in [the project README](../README.md). For the frontend alone:

```powershell
npm ci
npm run dev
```

Set `NEXT_PUBLIC_API_URL` to the Go API origin. The local example is in `.env.example`; for local development, create `.env.local` with `NEXT_PUBLIC_API_URL=http://localhost:8080`.

The form sends an `X-Idempotency-Key` with each new submission and reuses it when retrying the same payload.
