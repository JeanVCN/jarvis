export type OpenCodeStatus = { status: "connected" | "unavailable"; version?: string };
export type OpenCodeProject = { id: string; name: string };
export type OpenCodeSession = {
  id: string;
  title: string;
  updated_at: number;
  outcome: "succeeded" | "failed" | "interrupted" | null;
};

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { cache: "no-store" });
  if (!response.ok) throw new Error(response.status === 404 ? "bridge_disabled" : "opencode_unavailable");
  return (await response.json()) as T;
}

export function getOpenCodeStatus() {
  return getJSON<OpenCodeStatus>("/api/opencode/status");
}

export async function getOpenCodeProjects() {
  const result = await getJSON<{ projects: OpenCodeProject[] }>("/api/opencode/projects");
  return result.projects;
}

export async function getOpenCodeSessions(projectID: string) {
  const result = await getJSON<{ sessions: OpenCodeSession[] }>(
    `/api/opencode/projects/${encodeURIComponent(projectID)}/sessions`,
  );
  return result.sessions;
}
