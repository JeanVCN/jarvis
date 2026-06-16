export type HealthResponse = {
  status: "ok" | "degraded" | "offline";
  database: "ok" | "unavailable";
  service: string;
};
