import { createHash, createHmac, randomBytes, timingSafeEqual } from "node:crypto";

export const ACCESS_COOKIE = "relay_access";
const SESSION_TTL_MS = 12 * 60 * 60 * 1000;

export function isSiteAccessConfigured() {
  const password = process.env.SITE_ACCESS_PASSWORD || "";
  const secret = process.env.SITE_SESSION_SECRET || "";
  return password.length >= 16 && secret.length >= 32;
}

export function verifySiteAccessPassword(candidate: string) {
  if (!isSiteAccessConfigured()) return false;
  const expected = createHash("sha256").update(process.env.SITE_ACCESS_PASSWORD || "").digest();
  const supplied = createHash("sha256").update(candidate).digest();
  return timingSafeEqual(supplied, expected);
}

export function createSiteAccessSession() {
  const payload = Buffer.from(JSON.stringify({
    expiresAt: Date.now() + SESSION_TTL_MS,
    nonce: randomBytes(16).toString("base64url"),
  })).toString("base64url");
  return payload + "." + signature(payload);
}

export function verifySiteAccessSession(token?: string | null) {
  if (!isSiteAccessConfigured() || !token) return false;
  const separator = token.lastIndexOf(".");
  if (separator < 1) return false;

  const payload = token.slice(0, separator);
  const supplied = Buffer.from(token.slice(separator + 1), "base64url");
  const expected = Buffer.from(signature(payload), "base64url");
  if (supplied.length !== expected.length || !timingSafeEqual(supplied, expected)) return false;

  try {
    const session = JSON.parse(Buffer.from(payload, "base64url").toString()) as { expiresAt?: number };
    return typeof session.expiresAt === "number" && session.expiresAt > Date.now();
  } catch {
    return false;
  }
}

function signature(payload: string) {
  const secret = process.env.SITE_SESSION_SECRET;
  if (!secret || secret.length < 32) throw new Error("Site session signing is not configured");
  return createHmac("sha256", secret).update(payload).digest("base64url");
}
