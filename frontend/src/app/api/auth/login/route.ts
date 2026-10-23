import { randomBytes } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";

export async function GET(request: NextRequest) {
  const clientId = process.env.GITHUB_CLIENT_ID;
  if (!clientId) return new NextResponse("GitHub OAuth is not configured", { status: 503 });
  const state = randomBytes(32).toString("base64url");
  const site = process.env.PUBLIC_SITE_URL || request.nextUrl.origin;
  const callback = new URL("/api/auth/callback", site).toString();
  const authorize = new URL("https://github.com/login/oauth/authorize");
  authorize.searchParams.set("client_id", clientId);
  authorize.searchParams.set("redirect_uri", callback);
  authorize.searchParams.set("scope", "read:user");
  authorize.searchParams.set("state", state);
  const response = NextResponse.redirect(authorize);
  response.cookies.set("relay_oauth_state", state, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/api/auth/callback",
    maxAge: 600,
  });
  return response;
}
