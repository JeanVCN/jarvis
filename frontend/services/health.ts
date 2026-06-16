import type { HealthResponse } from "@/types/health";

const publicApiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const serverApiUrl = process.env.API_INTERNAL_URL ?? publicApiUrl;

export async function getHealth(): Promise<HealthResponse & { apiUrl: string }> {
  try {
    const response = await fetch(`${serverApiUrl}/health`, {
      cache: "no-store",
    });

    if (!response.ok) {
      return {
        status: "degraded",
        database: "unavailable",
        service: "jarvis-backend",
        apiUrl: publicApiUrl,
      };
    }

    const data = (await response.json()) as HealthResponse;

    return {
      ...data,
      apiUrl: publicApiUrl,
    };
  } catch {
    return {
      status: "offline",
      database: "unavailable",
      service: "jarvis-backend",
      apiUrl: publicApiUrl,
    };
  }
}
