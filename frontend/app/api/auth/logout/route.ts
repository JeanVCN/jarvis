import { NextResponse } from "next/server";
import { AUTH_COOKIE_NAME, authIsRequired, sameOriginRequest } from "@/lib/auth-session";

export async function POST(request: Request) {
  if (!authIsRequired()) return NextResponse.json({ error: "auth_not_required" }, { status: 404 });
  if (!sameOriginRequest(request)) return NextResponse.json({ error: "origin_denied" }, { status: 403 });

  const response = NextResponse.json({ authenticated: false });
  response.cookies.set(AUTH_COOKIE_NAME, "", {
    httpOnly: true,
    secure: process.env.JARVIS_AUTH_COOKIE_SECURE !== "false",
    sameSite: "strict",
    path: "/",
    maxAge: 0,
  });
  return response;
}
