import { NextRequest, NextResponse } from "next/server";
import { readSession, SESSION_COOKIE } from "@/lib/auth";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  const session = readSession(request);
  return NextResponse.json({ authenticated: Boolean(session), user: session ? { login: session.login, name: session.name, avatarUrl: session.avatarUrl } : null }, {
    headers: { "Cache-Control": "no-store" },
  });
}

export async function POST() {
  const response = NextResponse.json({ ok: true });
  response.cookies.set(SESSION_COOKIE, "", { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "lax", path: "/", maxAge: 0 });
  return response;
}
