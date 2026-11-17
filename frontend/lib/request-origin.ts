export function sameOriginRequest(request: Request): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return false;

  const host = request.headers.get("host")?.trim();
  const forwardedProto = request.headers.get("x-forwarded-proto")?.split(",", 1)[0]?.trim();
  const protocol = forwardedProto || new URL(request.url).protocol.replace(":", "");
  if (!host || !/^[a-z0-9.-]+(?::\d{1,5})?$/i.test(host) || !["http", "https"].includes(protocol)) return false;

  try {
    const parsed = new URL(origin);
    const expected = new URL(`${protocol}://${host}`);
    return parsed.origin === expected.origin;
  } catch {
    return false;
  }
}
