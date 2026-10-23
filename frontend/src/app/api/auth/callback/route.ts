import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE, signSession } from "@/lib/auth";

export const runtime = "nodejs";

export async function GET(request: NextRequest) {
  const site = process.env.PUBLIC_SITE_URL || request.nextUrl.origin;
  const code = request.nextUrl.searchParams.get("code");
  const state = request.nextUrl.searchParams.get("state");
  const savedState = request.cookies.get("relay_oauth_state")?.value;
  const clientId = process.env.GITHUB_CLIENT_ID;
  const clientSecret = process.env.GITHUB_CLIENT_SECRET;
  if (!code || !state || !savedState || state !== savedState || !clientId || !clientSecret) {
    return NextResponse.redirect(new URL("/?auth=failed", site));
  }

  try {
    const tokenResponse = await fetch("https://github.com/login/oauth/access_token", {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ client_id: clientId, client_secret: clientSecret, code }),
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
    const tokenData = (await tokenResponse.json()) as { access_token?: string };
    if (!tokenResponse.ok || !tokenData.access_token) throw new Error("OAuth token exchange failed");

    const userResponse = await fetch("https://api.github.com/user", {
      headers: { Authorization: `Bearer ${tokenData.access_token}`, Accept: "application/vnd.github+json", "User-Agent": "Cross-Team-Knowledge-Relay" },
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
    const user = (await userResponse.json()) as { login?: string; name?: string | null; avatar_url?: string };
    if (!userResponse.ok || !user.login) throw new Error("Could not read GitHub identity");
    const allowedUsers = process.env.GITHUB_ALLOWED_USERS?.split(",").map((value) => value.trim().toLowerCase()).filter(Boolean) || [];
    if (allowedUsers.length && !allowedUsers.includes(user.login.toLowerCase())) {
      return NextResponse.redirect(new URL("/?auth=not-allowed", site));
    }

    const now = Date.now();
    const session = signSession({
      login: user.login,
      name: user.name || user.login,
      avatarUrl: user.avatar_url || "",
      expiresAt: now + 12 * 60 * 60 * 1000,
    });
    const response = NextResponse.redirect(new URL("/", site));
    response.cookies.set(SESSION_COOKIE, session, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 12 * 60 * 60,
    });
    response.cookies.set("relay_oauth_state", "", { httpOnly: true, secure: process.env.NODE_ENV === "production", sameSite: "lax", path: "/api/auth/callback", maxAge: 0 });
    return response;
  } catch {
    return NextResponse.redirect(new URL("/?auth=failed", site));
  }
}
