import { NextRequest } from "next/server";
export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  const base = process.env.RELAY_API_URL?.replace(/\/$/, "");
  const token = process.env.RELAY_API_TOKEN;
  if (!base || !token) {
    return new Response("Bounty service is not configured", { status: 503 });
  }
  try {
    const upstream = await fetch(`${base}/api/v1/events`, {
      headers: { Authorization: `Bearer ${token}`, ...(request.headers.get("last-event-id") ? { "Last-Event-ID": request.headers.get("last-event-id")! } : {}) },
      cache: "no-store",
      signal: AbortSignal.timeout(60_000),
    });
    if (!upstream.ok || !upstream.body) {
      return new Response("Bounty event stream is unavailable", { status: upstream.status || 502 });
    }
    return new Response(upstream.body, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache, no-transform",
        Connection: "keep-alive",
        "X-Accel-Buffering": "no",
      },
    });
  } catch {
    return new Response("Bounty event stream is unavailable", { status: 502 });
  }
}
