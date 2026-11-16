import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { OpenCode } from "@opencode/client";
import { Service } from "@opencode/client/service";
import { loadAllowedProjects } from "./projects.js";

const port = Number(process.env.JARVIS_OPENCODE_BRIDGE_PORT ?? "18110");
const token = process.env.JARVIS_OPENCODE_BRIDGE_TOKEN ?? "";
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
      json(response, 200, { status: "connected", version: info.version });
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
  if (request.method === "GET" && sessionsMatch) {
    const project = projects.find(({ id }) => id === sessionsMatch[1]);
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

  json(response, 404, { error: "not_found" });
});

server.listen(port, "127.0.0.1", () => {
  console.log(`Jarvis OpenCode bridge listening on 127.0.0.1:${port}`);
});

for (const signal of ["SIGINT", "SIGTERM"] as const) {
  process.on(signal, () => server.close(() => process.exit(0)));
}
