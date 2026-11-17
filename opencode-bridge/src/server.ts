import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import { timingSafeEqual } from "node:crypto";
import { OpenCode } from "@opencode/client";
import { Service } from "@opencode/client/service";
import { loadAllowedProjects } from "./projects.js";
import { createProjectSession, findCreatedSession, interruptProjectSession, readProjectSession, replyProjectForm, replyProjectPermission, sendProjectPrompt, validOperationID, type InlineImage } from "./workflow.js";

const port = Number(process.env.JARVIS_OPENCODE_BRIDGE_PORT ?? "18110");
const token = process.env.JARVIS_OPENCODE_BRIDGE_TOKEN ?? "";
const writeEnabled = process.env.JARVIS_OPENCODE_WRITE_ENABLED === "true";
const projects = loadAllowedProjects(process.env.JARVIS_OPENCODE_PROJECTS);
const maxImageBytes = 10 * 1024 * 1024;
const maxImagePromptBodyBytes = 15 * 1024 * 1024;
const maxFormAnswerBodyBytes = 32 * 1024;

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

async function readBody(request: IncomingMessage, maxBytes = 12 * 1024): Promise<unknown> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of request) {
    const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += bytes.byteLength;
    if (size > maxBytes) throw new Error("request_too_large");
    chunks.push(bytes);
  }
  return JSON.parse(Buffer.concat(chunks).toString("utf8")) as unknown;
}

function parseInlineImage(value: unknown): InlineImage | undefined {
  if (value === undefined) return undefined;
  if (!value || typeof value !== "object") throw new Error("invalid_image");
  const image = value as Record<string, unknown>;
  const mime = image.mime;
  const base64 = image.base64;
  if ((mime !== "image/jpeg" && mime !== "image/png" && mime !== "image/webp") || typeof base64 !== "string" || !base64) {
    throw new Error("invalid_image");
  }
  if (base64.length > Math.ceil(maxImageBytes / 3) * 4 || base64.length % 4 !== 0) {
    throw new Error("invalid_image");
  }
  let padding = 0;
  for (let i = 0; i < base64.length; i++) {
    const code = base64.charCodeAt(i);
    const isLetter = (code >= 65 && code <= 90) || (code >= 97 && code <= 122);
    const isDigit = code >= 48 && code <= 57;
    if (isLetter || isDigit || code === 43 || code === 47) {
      if (padding > 0) throw new Error("invalid_image");
      continue;
    }
    if (code === 61 && i >= base64.length - 2 && padding < 2) {
      padding++;
      continue;
    }
    throw new Error("invalid_image");
  }
  const bytes = Buffer.from(base64, "base64");
  if (bytes.length === 0 || bytes.length > maxImageBytes || bytes.toString("base64") !== base64) {
    throw new Error("invalid_image");
  }
  const signatureMatches = mime === "image/jpeg"
    ? bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff
    : mime === "image/png"
      ? bytes.length >= 8 && bytes.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]))
      : bytes.length >= 12 && bytes.toString("ascii", 0, 4) === "RIFF" && bytes.toString("ascii", 8, 12) === "WEBP";
  if (!signatureMatches) throw new Error("invalid_image");
  return { mime, base64 };
}

function projectByID(id: string) {
  return projects.find((project) => project.id === id);
}

function isFormAnswer(value: unknown): value is Record<string, string | number | boolean | string[]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const entries = Object.entries(value);
  return entries.length > 0 && entries.length <= 16 && entries.every(([, answer]) =>
    typeof answer === "string" || typeof answer === "number" || typeof answer === "boolean" ||
    (Array.isArray(answer) && answer.length <= 32 && answer.every((item) => typeof item === "string")),
  );
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
      body = await readBody(request, maxImagePromptBodyBytes);
    } catch (error) {
      json(response, error instanceof Error && error.message === "request_too_large" ? 413 : 400, { error: "invalid_request" });
      return;
    }
    const candidate = body && typeof body === "object" ? body as Record<string, unknown> : {};
    if (!validOperationID(candidate.operation_id) || typeof candidate.text !== "string" || !candidate.text.trim() || candidate.text.length > 8000) {
      json(response, 400, { error: "invalid_prompt" });
      return;
    }
    let image: InlineImage | undefined;
    try {
      image = parseInlineImage(candidate.image);
    } catch {
      json(response, 400, { error: "invalid_image" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      json(response, 202, await sendProjectPrompt(client, project, decodeURIComponent(promptMatch[2]), candidate.operation_id, candidate.text, image));
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

  const formMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions\/([^/]+)\/forms\/([^/]+)\/reply$/);
  if (request.method === "POST" && formMatch) {
    if (!writeEnabled) {
      json(response, 403, { error: "opencode_write_disabled" });
      return;
    }
    const project = projectByID(formMatch[1]);
    if (!project) {
      json(response, 404, { error: "project_not_found" });
      return;
    }
    let body: unknown;
    try {
      body = await readBody(request, maxFormAnswerBodyBytes);
    } catch (error) {
      json(response, error instanceof Error && error.message === "request_too_large" ? 413 : 400, { error: "invalid_request" });
      return;
    }
    const candidate = body && typeof body === "object" ? body as Record<string, unknown> : {};
    if (!validOperationID(candidate.operation_id) || !isFormAnswer(candidate.answer)) {
      json(response, 400, { error: "invalid_form_answer" });
      return;
    }
    try {
      const client = await openCodeClient();
      if (!client) {
        json(response, 503, { error: "opencode_unavailable" });
        return;
      }
      await replyProjectForm(client, project, decodeURIComponent(formMatch[2]), decodeURIComponent(formMatch[3]), candidate.answer);
      json(response, 200, { status: "completed" });
    } catch (error) {
      if (error instanceof Error && (error.message === "session_not_in_project" || error.message === "form_not_pending")) {
        json(response, 404, { error: "form_not_pending" });
        return;
      }
      json(response, 502, { error: "opencode_request_failed" });
    }
    return;
  }

  const interruptMatch = url.pathname.match(/^\/v1\/projects\/([a-z0-9_-]+)\/sessions\/([^/]+)\/interrupt$/);
  if (request.method === "POST" && interruptMatch) {
    if (!writeEnabled) {
      json(response, 403, { error: "opencode_write_disabled" });
      return;
    }
    const project = projectByID(interruptMatch[1]);
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
      await interruptProjectSession(client, project, decodeURIComponent(interruptMatch[2]));
      json(response, 202, { status: "interrupt_requested" });
    } catch (error) {
      if (error instanceof Error && error.message === "session_not_in_project") {
        json(response, 404, { error: "session_not_found" });
        return;
      }
      if (error instanceof Error && error.message === "session_not_running") {
        json(response, 409, { error: "session_not_running" });
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
