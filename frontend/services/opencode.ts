export type OpenCodeStatus = {
  status: "connected" | "unavailable";
  version?: string;
  write_enabled?: boolean;
};
export type OpenCodeProject = { id: string; name: string };
export type OpenCodeSession = {
  id: string;
  title: string;
  updated_at: number;
  outcome: "succeeded" | "failed" | "interrupted" | null;
};
export type JarvisSession = {
  id: string;
  project_id: string;
  external_id?: string;
  title: string;
  state: "creating" | "ready" | "uncertain" | "failed";
  updated_at: number;
};
export type OpenCodeOperation = {
  id: string;
  project_id: string;
  session_id: string;
  kind: "create_session" | "prompt" | "permission";
  input?: string;
  state: "submitting" | "accepted" | "running" | "completed" | "failed" | "uncertain";
  external_id?: string;
  error_code?: string;
};
export type SessionDetail = {
  session: OpenCodeSession & { state: "idle" | "running" };
  messages: Array<{
    id: string;
    role: "user" | "assistant";
    content: string;
    created_at: number;
    completed: boolean;
  }>;
  diff: Array<{
    file: string;
    patch: string;
    additions: number;
    deletions: number;
    status: string;
    truncated?: boolean;
  }>;
  permissions: Array<{ id: string; action: string; resources: string[]; message?: string }>;
};
export type JarvisSessionSnapshot = {
  session: JarvisSession;
  detail: SessionDetail | null;
  operations: OpenCodeOperation[];
};

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { cache: "no-store" });
  if (!response.ok) throw new Error(response.status === 404 ? "not_found" : "opencode_unavailable");
  return (await response.json()) as T;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = (await response.json().catch(() => ({}))) as { error?: string } & T;
  if (!response.ok && response.status !== 202) {
    throw new Error(payload.error ?? "opencode_request_failed");
  }
  return payload;
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

export async function getJarvisSessions(projectID: string) {
  const result = await getJSON<{ sessions: JarvisSession[] }>(
    `/api/opencode/projects/${encodeURIComponent(projectID)}/jarvis-sessions`,
  );
  return result.sessions;
}

export async function getJarvisSession(sessionID: string) {
  return getJSON<JarvisSessionSnapshot>(`/api/opencode/sessions/${encodeURIComponent(sessionID)}`);
}

export async function createJarvisSession(projectID: string, operationID: string) {
  return postJSON<{ operation: OpenCodeOperation }>(
    `/api/opencode/projects/${encodeURIComponent(projectID)}/sessions`,
    { operation_id: operationID },
  );
}

export async function sendOpenCodePrompt(sessionID: string, operationID: string, text: string) {
  return postJSON<{ operation: OpenCodeOperation }>(
    `/api/opencode/sessions/${encodeURIComponent(sessionID)}/prompts`,
    { operation_id: operationID, text },
  );
}

export async function replyOpenCodePermission(
  sessionID: string,
  requestID: string,
  operationID: string,
  decision: "once" | "reject",
) {
  return postJSON<{ operation: OpenCodeOperation }>(
    `/api/opencode/sessions/${encodeURIComponent(sessionID)}/permissions/${encodeURIComponent(requestID)}/reply`,
    { operation_id: operationID, decision },
  );
}
