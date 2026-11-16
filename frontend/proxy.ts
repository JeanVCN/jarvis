import { NextResponse, type NextRequest } from "next/server";
import {
  AUTH_COOKIE_NAME,
  authConfig,
  authIsRequired,
  sameOriginRequest,
  verifySessionToken,
} from "@/lib/auth-session";

const publicPaths = new Set(["/login", "/api/auth/login"]);

export async function proxy(request: NextRequest) {
  if (!authIsRequired()) return NextResponse.next();

  const auth = authConfig();
  if (!auth) {
    return request.nextUrl.pathname.startsWith("/api/")
      ? NextResponse.json({ error: "auth_not_configured" }, { status: 503 })
      : new NextResponse("Jarvis authentication is required but not configured", { status: 503 });
  }

  if (publicPaths.has(request.nextUrl.pathname)) return NextResponse.next();

  if (!["GET", "HEAD", "OPTIONS"].includes(request.method) && !sameOriginRequest(request)) {
    return NextResponse.json({ error: "origin_denied" }, { status: 403 });
  }

  const token = request.cookies.get(AUTH_COOKIE_NAME)?.value;
  if (verifySessionToken(token, auth.secret)) return NextResponse.next();

  if (request.nextUrl.pathname.startsWith("/api/")) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  const loginURL = request.nextUrl.clone();
  loginURL.pathname = "/login";
  loginURL.search = "";
  loginURL.searchParams.set("next", `${request.nextUrl.pathname}${request.nextUrl.search}`);
  return NextResponse.redirect(loginURL);
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
