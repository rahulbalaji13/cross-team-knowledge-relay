import { NextRequest, NextResponse } from "next/server";
import { readSession } from "@/lib/auth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function backendConfig() {
  const base = process.env.RELAY_API_URL?.replace(/\/$/, "");
  const token = process.env.RELAY_API_TOKEN;
  if (!base || !token) throw new Error("Bounty API is not configured");
  return { base, token };
}

export async function GET() {
  try {
    const { base, token } = backendConfig();
    const upstream = await fetch(`${base}/api/v1/bounties`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
      signal: AbortSignal.timeout(8000),
    });
    return new NextResponse(await upstream.text(), {
      status: upstream.status,
      headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json({ error: "Bounty service is unavailable" }, { status: 503 });
  }
}

export async function POST(request: NextRequest) {
  if (request.headers.get("origin") !== request.nextUrl.origin) {
    return NextResponse.json({ error: "Invalid request origin" }, { status: 403 });
  }
  const session = readSession(request);
  if (!session) return NextResponse.json({ error: "Sign in with GitHub to post a bounty" }, { status: 401 });

  try {
    const { base, token } = backendConfig();
    const body = await request.text();
    if (new TextEncoder().encode(body).byteLength > 64 * 1024) {
      return NextResponse.json({ error: "Request body is too large" }, { status: 413 });
    }
    let payload: unknown;
    try { payload = JSON.parse(body); } catch {
      return NextResponse.json({ error: "Invalid JSON request" }, { status: 400 });
    }
    if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
      return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
    }
    const authenticatedBody = { ...(payload as Record<string, unknown>), created_by: session.login };
    const upstream = await fetch(`${base}/api/v1/bounties`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
      body: JSON.stringify(authenticatedBody),
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
    return new NextResponse(await upstream.text(), {
      status: upstream.status,
      headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
    });
  } catch {
    return NextResponse.json({ error: "Bounty service is unavailable" }, { status: 503 });
  }
}
