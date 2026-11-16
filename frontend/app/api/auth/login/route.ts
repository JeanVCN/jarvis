import { createHmac } from "node:crypto";
import { NextResponse } from "next/server";
import {
  AUTH_COOKIE_NAME,
  AUTH_SESSION_SECONDS,
  authConfig,
  authIsRequired,
  safePasswordEqual,
  sameOriginRequest,
} from "@/lib/auth-session";

let failedAttempts = 0;
let failureWindowStartedAt = Date.now();
const failureWindowMs = 15 * 60 * 1000;
const maxFailedAttempts = 10;
const maxRequestBytes = 4096;

async function readLoginBody(request: Request): Promise<unknown> {
  const reader = request.body?.getReader();
  if (!reader) throw new Error("invalid_request");
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > maxRequestBytes) {
        await reader.cancel();
        throw new Error("request_too_large");
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return JSON.parse(new TextDecoder().decode(bytes)) as unknown;
}

export async function POST(request: Request) {
  if (!authIsRequired()) {
    return NextResponse.json({ error: "auth_not_required" }, { status: 404 });
  }
  const auth = authConfig();
  if (!auth) return NextResponse.json({ error: "auth_not_configured" }, { status: 503 });
  if (!sameOriginRequest(request)) return NextResponse.json({ error: "origin_denied" }, { status: 403 });

  const now = Date.now();
  if (now - failureWindowStartedAt >= failureWindowMs) {
    failureWindowStartedAt = now;
    failedAttempts = 0;
  }
  if (failedAttempts >= maxFailedAttempts) {
    return NextResponse.json({ error: "too_many_attempts" }, { status: 429 });
  }

  let body: unknown;
  try {
    body = await readLoginBody(request);
  } catch (error) {
    if (error instanceof Error && error.message === "request_too_large") {
      return NextResponse.json({ error: "request_too_large" }, { status: 413 });
    }
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  const password =
    body && typeof body === "object" && "password" in body && typeof body.password === "string"
      ? body.password
      : "";
  if (!password || password.length > 1024 || !safePasswordEqual(password, auth.password)) {
    failedAttempts += 1;
    return NextResponse.json({ error: "invalid_credentials" }, { status: 401 });
  }

  failedAttempts = 0;
  const expiresAt = Math.floor(now / 1000) + AUTH_SESSION_SECONDS;
  const payload = `v1.${expiresAt}`;
  const signature = createHmac("sha256", auth.secret).update(payload).digest("base64url");
  const response = NextResponse.json({ authenticated: true });
  response.cookies.set(AUTH_COOKIE_NAME, `${payload}.${signature}`, {
    httpOnly: true,
    secure: process.env.JARVIS_AUTH_COOKIE_SECURE !== "false",
    sameSite: "strict",
    path: "/",
    maxAge: AUTH_SESSION_SECONDS,
  });
  return response;
}
