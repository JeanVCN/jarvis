import { NextResponse, type NextRequest } from "next/server";
import { sameOriginRequest } from "@/lib/request-origin";

export async function proxy(request: NextRequest) {
  if (!["GET", "HEAD", "OPTIONS"].includes(request.method) && !sameOriginRequest(request)) {
    return NextResponse.json({ error: "origin_denied" }, { status: 403 });
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
