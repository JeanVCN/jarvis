import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { OpenCode } from "@opencode/client";
import { Service } from "@opencode/client/service";
import { loadAllowedProjects } from "./projects.js";
import { createProjectSession, findCreatedSession, readProjectSession, replyProjectPermission, sendProjectPrompt, validOperationID } from "./workflow.js";

const port = Number(process.env.JARVIS_OPENCODE_BRIDGE_PORT ?? "18110");
const token = process.env.JARVIS_OPENCODE_BRIDGE_TOKEN ?? "";
const writeEnabled = process.env.JARVIS_OPENCODE_WRITE_ENABLED === "true";
const projects = loadAllowedProjects(process.env.JARVIS_OPENCODE_PROJECTS);

if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error("JARVIS_OPENCODE_BRIDGE_PORT must be a valid TCP port");
}
if (Buffer.byteLength(token) < 32) {
  throw new Error("JARVIS_OPENCODE_BRIDGE_TOKEN must contain at least 32 bytes");
}

function authorized(request: IncomingMessage): boolean {
  const header = request.headers.authorization ?? "";
  const supplied = Buffer.from(header.startsWith("Bearer ") ? header.slice(7) : "");
  const expected = Buffer.from(token);
  return supplied.length === expected.length && timingSafeEqual(supplied, expected);
}

function json(response: ServerResponse, status: number, body: unknown): void {
  response.writeHead(status, {
    "content-type": "application/json; charset=utf-8",
    "cache-control": "no-store",
    "x-content-type-options": "nosniff",
  });
  response.end(JSON.stringify(body));
}

async function openCodeClient() {
  const endpoint = await Service.discover({ version: (value) => value.startsWith("2.") });
  if (!endpoint) return undefined;
  return OpenCode.make({
    baseUrl: endpoint.url,
    headers: Service.headers(endpoint),
  });
}

async function readBody(request: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of request) {
    const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += bytes.byteLength;
    if (size > 12 * 1024) throw new Error("request_too_large");
    chunks.push(bytes);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8")) as unknown;
}

function projectByID(id: string) {
  return projects.find((project) => project.id === id);
}

const server = createServer(async (request, response) => {
  const url = new URL(request.url ?? "/", "http://127.0.0.1");

  if (request.method === "GET" && url.pathname === "/healthz") {
    json(response, 200, { status: "ok" });
    return;
  }

  if (!authorized(request)) {
    json(response, 401, { error: "unauthorized" });
    return;
  }

  if (request.method === "GET" && url.pathname === "/v1/status") {
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { status: "unavailable", error: "opencode_unavailable" });
        return;
      }
      const info = await client.server.info();
      json(response, 200, { status: "connected", version: info.version, write_enabled: writeEnabled });
    } catch {
      json(response, 503, { status: "unavailable", error: "opencode_unavailable" });
    }
    return;
  }

  if (request.method === "GET" && url.pathname === "/v1/projects") {
    json(response, 200, {
      projects: projects.map(({ id, name }) => ({ id, name })),
    });
    return;
  }

  const sessionsMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions$/);
  if (request.method === "POST" && sessionsMatch) {
    if (!writeEnabled) {
      json(response, 403, { error: "opencode_write_disabled" });
      return;
    }
    const project = projectByID(sessionsMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    let body: unknown;
    try {
      body = await readBody(request);
    } catch (error) {
      json(response, error instanceof Error && error.message === "request_too_large" ? 413 : 400, { error: "invalid_request" });
      return;
    }
    const operationID = body && typeof body === "object" && "operation_id" in body ? body.operation_id : undefined;
    if (!validOperationID(operationID)) {
      json(response, 400, { error: "invalid_operation_id" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      json(response, 201, await createProjectSession(client, project, operationID));
    } catch {
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  if (request.method === "GET" && sessionsMatch) {
    const project = projectByID(sessionsMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      const result = await client.session.list({ directory: project.directory, limit: 50 });
      json(response, 200, {
        sessions: result.data.map((session) => ({
          id: session.id,
          title: session.title ?? "Sem título",
          updated_at: session.time.updated,
          outcome: session.outcome ?? null,
        })),
      });
    } catch {
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  const operationMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/operations\/([a-f0-9]{32})$/);
  if (request.method === "GET" && operationMatch) {
    const project = projectByID(operationMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      const session = await findCreatedSession(client, project, operationMatch[2]);
      json(response, 200, { session: session ?? null });
    } catch {
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  const promptMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions\/([^/]+)\/prompts$/);
  if (request.method === "POST" && promptMatch) {
    if (!writeEnabled) {
      json(response, 403, { error: "opencode_write_disabled" });
      return;
    }
    const project = projectByID(promptMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    let body: unknown;
    try {
      body = await readBody(request);
    } catch (error) {
      json(response, error instanceof Error && error.message === "request_too_large" ? 413 : 400, { error: "invalid_request" });
      return;
    }
    const candidate = body && typeof body === "object" ? body as Record<string, unknown> : {};
    if (!validOperationID(candidate.operation_id) || typeof candidate.text !== "string" || !candidate.text.trim() || candidate.text.length > 8000) {
      json(response, 400, { error: "invalid_prompt" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      json(response, 202, await sendProjectPrompt(client, project, decodeURIComponent(promptMatch[2]), candidate.operation_id, candidate.text));
    } catch (error) {
      if (error instanceof Error && error.message === "session_not_in_project") {
        json(response, 404, { error: "session_not_found" });
        return;
      }
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  const permissionMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions\/([^/]+)\/permissions\/([^/]+)\/reply$/);
  if (request.method === "POST" && permissionMatch) {
    if (!writeEnabled) {
      json(response, 403, { error: "opencode_write_disabled" });
      return;
    }
    const project = projectByID(permissionMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    let body: unknown;
    try {
      body = await readBody(request);
    } catch (error) {
      json(response, error instanceof Error && error.message === "request_too_large" ? 413 : 400, { error: "invalid_request" });
      return;
    }
    const candidate = body && typeof body === "object" ? body as Record<string, unknown> : {};
    const operationID = candidate.operation_id;
    const decision = candidate.decision;
    if (!validOperationID(operationID) || (decision !== "once" && decision !== "reject")) {
      json(response, 400, { error: "invalid_permission_decision" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      await replyProjectPermission(client, project, decodeURIComponent(permissionMatch[2]), decodeURIComponent(permissionMatch[3]), decision);
      json(response, 200, { status: "completed" });
    } catch (error) {
      if (error instanceof Error && (error.message === "session_not_in_project" || error.message === "permission_not_pending")) {
        json(response, 404, { error: "permission_not_pending" });
        return;
      }
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  const sessionMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions\/([^/]+)$/);
  if (request.method === "GET" && sessionMatch) {
    const project = projectByID(sessionMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      json(response, 200, await readProjectSession(client, project, decodeURIComponent(sessionMatch[2])));
    } catch (error) {
      if (error instanceof Error && error.message === "session_not_in_project") {
        json(response, 404, { error: "session_not_found" });
        return;
      }
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  json(response, 404, { error: "not_found" });
});

server.listen(port, "127.0.0.1", () => {
  console.log(`Jarvis OpenCode bridge listening on 127.0.0.1:${port}`);
});

for (const signal of ["SIGINT", "SIGTERM"] as const) {
  process.on(signal, () => server.close(() => process.exit(0)));
}
