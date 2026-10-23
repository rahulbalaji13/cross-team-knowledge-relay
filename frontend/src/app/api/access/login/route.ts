import { NextRequest, NextResponse } from "next/server";
import { ACCESS_COOKIE, createSiteAccessSession, isSiteAccessConfigured, verifySiteAccessPassword } from "@/lib/site-access";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function POST(request: NextRequest) {
  if (request.headers.get("origin") !== request.nextUrl.origin) {
    return NextResponse.json({ error: "Invalid request origin" }, { status: 403 });
  }
  if (!isSiteAccessConfigured()) {
    return NextResponse.json({ error: "Site access has not been configured by the owner." }, { status: 503 });
  }

  let body: string;
  try {
    body = await request.text();
  } catch {
    return NextResponse.json({ error: "Invalid request" }, { status: 400 });
  }
  if (new TextEncoder().encode(body).byteLength > 4096) {
    return NextResponse.json({ error: "Request is too large" }, { status: 413 });
  }

  let payload: unknown;
  try {
    payload = JSON.parse(body);
  } catch {
    return NextResponse.json({ error: "Invalid request" }, { status: 400 });
  }
  const password = payload && typeof payload === "object" && !Array.isArray(payload)
    ? (payload as Record<string, unknown>).password
    : null;
  if (typeof password !== "string" || password.length > 1024 || !verifySiteAccessPassword(password)) {
    return NextResponse.json({ error: "Incorrect password." }, { status: 401, headers: { "Cache-Control": "no-store" } });
  }

  const response = NextResponse.json({ ok: true }, { headers: { "Cache-Control": "no-store" } });
  response.cookies.set(ACCESS_COOKIE, createSiteAccessSession(), {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "strict",
    path: "/",
    maxAge: 12 * 60 * 60,
  });
  return response;
}
