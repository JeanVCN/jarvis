import { createHmac, timingSafeEqual } from "node:crypto";

export const AUTH_COOKIE_NAME = "jarvis_session";
export const AUTH_SESSION_SECONDS = 12 * 60 * 60;

export function authIsRequired(): boolean {
  return process.env.JARVIS_AUTH_REQUIRED === "true";
}

export function authConfig(): { password: string; secret: string } | undefined {
  const password = process.env.JARVIS_AUTH_PASSWORD ?? "";
  const secret = process.env.JARVIS_AUTH_SECRET ?? "";
  if (password.length < 12 || Buffer.byteLength(secret) < 32) return undefined;
  return { password, secret };
}

export function safePasswordEqual(supplied: string, expected: string): boolean {
  const suppliedDigest = createHmac("sha256", "jarvis-password-check").update(supplied).digest();
  const expectedDigest = createHmac("sha256", "jarvis-password-check").update(expected).digest();
  return timingSafeEqual(suppliedDigest, expectedDigest);
}

export function createSessionToken(secret: string, now = Date.now()): string {
  const expiresAt = Math.floor(now / 1000) + AUTH_SESSION_SECONDS;
  const payload = `v1.${expiresAt}`;
  const signature = createHmac("sha256", secret).update(payload).digest("base64url");
  return `${payload}.${signature}`;
}

export function verifySessionToken(
  token: string | undefined,
  secret: string,
  now = Date.now(),
): boolean {
  if (!token || token.length > 256) return false;
  const [version, expiresText, suppliedSignature, ...rest] = token.split(".");
  if (version !== "v1" || !expiresText || !suppliedSignature || rest.length > 0) return false;
  if (!/^\d{10}$/.test(expiresText)) return false;
  const expiresAt = Number(expiresText);
  if (!Number.isSafeInteger(expiresAt) || expiresAt <= Math.floor(now / 1000)) return false;

  const payload = `${version}.${expiresText}`;
  const expectedSignature = createHmac("sha256", secret).update(payload).digest();
  let signature: Buffer;
  try {
    signature = Buffer.from(suppliedSignature, "base64url");
  } catch {
    return false;
  }
  return signature.length === expectedSignature.length && timingSafeEqual(signature, expectedSignature);
}

export function sameOriginRequest(request: Request): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return false;

  const forwardedHost = request.headers.get("x-forwarded-host")?.split(",", 1)[0]?.trim();
  const host = forwardedHost || request.headers.get("host");
  const forwardedProto = request.headers.get("x-forwarded-proto")?.split(",", 1)[0]?.trim();
  const protocol = forwardedProto || new URL(request.url).protocol.replace(":", "");
  if (!host || !/^[a-z0-9.-]+(?::\d{1,5})?$/i.test(host)) return false;

  try {
    const parsed = new URL(origin);
    return parsed.origin === `${protocol}://${host}`;
  } catch {
    return false;
  }
}
