# 🚀 Easy Deployment Guide: Vercel (Frontend) & Render (Backend)

Because this is a "monorepo" (both frontend and backend folders are in the same GitHub repository), you need to tell Vercel and Render which folder to look at. Here is the foolproof, step-by-step guide to deploying without errors.

---

## Part 1: Prerequisites
1. **GitHub Repository**: Push your entire `cross-team-match` folder to a GitHub repository.
2. **Databases (Free Tier)**:
   - **PostgreSQL**: Create a free database on [Render](https://dashboard.render.com).
   - **Redis**: Create a free Redis instance on [Upstash](https://upstash.com).
   - **Neo4j**: Create a free AuraDB instance on [Neo4j Aura](https://console.neo4j.io/).

---

## Part 2: Deploying the Go Backend on Render

Render is perfect for Go applications because it automatically detects Go code and builds it.

### Steps:
1. Go to [Render Dashboard](https://dashboard.render.com/).
2. Click **New +** and select **Web Service**.
3. Connect your GitHub account and select your repository.
4. Fill in the deployment settings:
   - **Name**: `cross-team-match-api` (or whatever you prefer)
   - **Root Directory**: `backend` *(⚠️ CRITICAL: If you leave this blank, the build will fail!)*
   - **Environment**: `Go`
   - **Build Command**: `go build -o server ./cmd/server/main.go`
   - **Start Command**: `./server`
5. Scroll down to **Advanced** -> **Environment Variables** and add your Database URLs:
   - `PORT` = `8080`
   - `NEO4J_URI` = `bolt://...` (From Neo4j Aura)
   - `NEO4J_AUTH` = `neo4j/your_password`
   - `REDIS_URL` = `redis://...` (From Upstash)
   - `POSTGRES_URL` = `postgres://...` (From Render Postgres)
6. Click **Create Web Service**.

> 🎉 **Result**: Render will build your Go backend and give you an API URL like: `https://cross-team-match-api.onrender.com`. Copy this URL for the frontend!

---

## Part 3: Deploying the Next.js Frontend on Vercel

Vercel is the creator of Next.js, so deployment is almost entirely automated.

### Steps:
1. Go to [Vercel](https://vercel.com/) and click **Add New** -> **Project**.
2. Import your GitHub repository.
3. In the "Configure Project" screen, look for **Root Directory**.
4. Click **Edit**, select the `frontend` folder, and save. *(⚠️ CRITICAL: Vercel must know the app is inside this folder!)*
5. Vercel will auto-detect "Next.js" as the Framework Preset. Leave the build commands as default.
6. Open the **Environment Variables** section and add:
   - `NEXT_PUBLIC_WS_URL` = `https://cross-team-match-api.onrender.com` (Paste the URL Render gave you in Part 2).
7. Click **Deploy**.

> 🎉 **Result**: Vercel will install your React packages, build your Next.js frontend, and give you a live HTTPS URL in under 2 minutes.

---

## 🛠️ Common Errors & How to Avoid Them

1. ❌ **"go: go.mod file not found in current directory" (Render)**
   - **Fix**: You forgot to set the `Root Directory` to `backend` in Render. Your `go.mod` is inside the `backend` folder, not the main repo folder.

2. ❌ **"Could not find a valid build in the '.next' directory" (Vercel)**
   - **Fix**: You forgot to set the `Root Directory` to `frontend` in Vercel.

3. ❌ **Frontend gets "Network Error" trying to reach the API**
   - **Fix**: Ensure your `NEXT_PUBLIC_WS_URL` on Vercel starts with `https://` and does **not** have a trailing slash (`/`).
