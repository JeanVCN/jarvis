import { realpathSync } from "node:fs";
import path from "node:path";
import type { OpenCodeClient } from "@opencode/client";
import type { AllowedProject } from "./projects.js";

const operationIDPattern = /^[a-f0-9]{32}$/;
const maxDiffBytes = 64 * 1024;
export type InlineImage = { mime: "image/jpeg" | "image/png" | "image/webp"; base64: string };

export type SafeSession = {
  id: string;
  title: string;
  updated_at: number;
  outcome: "succeeded" | "failed" | "interrupted" | null;
  state: "idle" | "running";
};

export function validOperationID(value: unknown): value is string {
  return typeof value === "string" && operationIDPattern.test(value);
}

export function safeDiffPath(value: string): string {
  const normalized = value.replaceAll("\\", "/");
  if (path.isAbsolute(value) || /^[a-z]:\//i.test(normalized) || normalized.startsWith("//") || normalized.split("/").includes("..")) {
    return path.posix.basename(normalized);
  }
  return normalized;
}

export async function assertSessionProject(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
): Promise<Awaited<ReturnType<OpenCodeClient["session"]["get"]>>> {
  const session = await client.session.get({ sessionID });
  let directory: string;
  try {
    directory = realpathSync(session.location.directory);
  } catch {
    throw new Error("session_not_in_project");
  }
  if (directory !== project.directory) throw new Error("session_not_in_project");
  return session;
}

export async function createProjectSession(
  client: OpenCodeClient,
  project: AllowedProject,
  operationID: string,
): Promise<SafeSession> {
  const session = await client.session.create({
    location: { directory: project.directory },
    permissions: [{ action: "*", resource: "*", effect: "ask" }],
    metadata: { jarvis_operation_id: operationID },
  });
  return {
    id: session.id,
    title: session.title ?? "Nova sessão",
    updated_at: session.time.updated,
    outcome: session.outcome ?? null,
    state: session.time.busy ? "running" : "idle",
  };
}

export async function findCreatedSession(
  client: OpenCodeClient,
  project: AllowedProject,
  operationID: string,
): Promise<SafeSession | undefined> {
  const result = await client.session.list({ directory: project.directory, limit: 100 });
  const matches = result.data.filter((session) => session.metadata?.jarvis_operation_id === operationID);
  if (matches.length !== 1) return undefined;
  // A directory-filtered listing identifies candidates. Verify ownership
  // from the session itself before restoring a persisted Jarvis link.
  const session = await assertSessionProject(client, project, matches[0].id);
  return {
    id: session.id,
    title: session.title ?? "Nova sessão",
    updated_at: session.time.updated,
    outcome: session.outcome ?? null,
    state: session.time.busy ? "running" : "idle",
  };
}

export async function sendProjectPrompt(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
  text: string,
  image?: InlineImage,
): Promise<{ inbox_id: string }> {
  await assertSessionProject(client, project, sessionID);
  const files = image ? [{
    uri: `data:${image.mime};base64,${image.base64}`,
    source: { type: "inline" as const },
    name: "whatsapp-image",
  }] : undefined;
  const accepted = await client.session.prompt({ sessionID, text, ...(files ? { files } : {}) });
  return { inbox_id: accepted.id };
}

export async function replyProjectPermission(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
  requestID: string,
  decision: "once" | "reject",
): Promise<void> {
  await assertSessionProject(client, project, sessionID);
  const request = await client.permission.get({ sessionID, requestID });
  if (request.id !== requestID || request.sessionID !== sessionID) throw new Error("permission_not_pending");
  await client.permission.reply({ sessionID, requestID, decision });
}

export async function replyProjectForm(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
  formID: string,
  answer: Record<string, string | number | boolean | string[]>,
): Promise<void> {
  await assertSessionProject(client, project, sessionID);
  const form = await client.session.form.get({ sessionID, formID });
  if (form.id !== formID || form.sessionID !== sessionID || form.state.status !== "pending") {
    throw new Error("form_not_pending");
  }
  const fields = new Map(form.fields.map((field) => [field.key, field]));
  for (const [key, value] of Object.entries(answer)) {
    const field = fields.get(key);
    if (!field) throw new Error("invalid_form_answer");
    const valid = field.type === "string"
      ? typeof value === "string"
      : field.type === "multiselect"
        ? Array.isArray(value) && value.every((item) => typeof item === "string")
        : field.type === "integer"
          ? typeof value === "number" && Number.isInteger(value)
          : field.type === "number"
            ? typeof value === "number" && Number.isFinite(value)
            : field.type === "boolean"
              ? typeof value === "boolean"
              : false;
    if (!valid) throw new Error("invalid_form_answer");
  }
  await client.session.form.reply({ sessionID, formID, answer });
}

export async function interruptProjectSession(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
): Promise<void> {
  const session = await assertSessionProject(client, project, sessionID);
  if (!session.time.busy) throw new Error("session_not_running");
  await client.session.interrupt({ sessionID });
}

export async function readProjectSession(
  client: OpenCodeClient,
  project: AllowedProject,
  sessionID: string,
) {
  const session = await assertSessionProject(client, project, sessionID);
  const [messages, diff, permissions, forms] = await Promise.all([
    client.message.list({ sessionID, limit: 100, order: "desc" }),
    client.session.diff({ sessionID }),
    client.permission.list({ sessionID }),
    pendingProjectForms(client, sessionID),
  ]);

  const publicMessages = [...messages.data].reverse().flatMap((message) => {
    if (message.type === "user") {
      return [{ id: message.id, role: "user", content: message.text, created_at: message.time.created, completed: true }];
    }
    if (message.type === "assistant") {
      const content = message.content
        .filter((part) => part.type === "text")
        .map((part) => part.text)
        .join("");
      if (!content) return [];
      return [{
        id: message.id,
        role: "assistant",
        content,
        created_at: message.time.created,
        completed: Boolean(message.time.completed),
        finish: message.finish ?? null,
      }];
    }
    return [];
  });

  let remainingDiffBytes = maxDiffBytes;
  const publicDiff = diff.slice(0, 100).map((file) => {
    const patch = Buffer.from(file.patch).subarray(0, remainingDiffBytes).toString();
    remainingDiffBytes = Math.max(0, remainingDiffBytes - Buffer.byteLength(patch));
    return {
      file: safeDiffPath(file.file),
      patch,
      additions: file.additions,
      deletions: file.deletions,
      status: file.status,
      truncated: patch.length < file.patch.length || remainingDiffBytes === 0,
    };
  });

  return {
    session: {
      id: session.id,
      title: session.title ?? "Sem título",
      updated_at: session.time.updated,
      outcome: session.outcome ?? null,
      state: session.time.busy ? "running" : "idle",
    },
    messages: publicMessages,
    diff: publicDiff,
    permissions: permissions.map((permission) => ({
      id: permission.id,
      action: permission.action,
      resources: permission.resources.map(safeDiffPath),
      message: permission.message,
    })),
    forms,
  };
}

async function pendingProjectForms(client: OpenCodeClient, sessionID: string) {
  const forms = await client.session.form.list({ sessionID });
  const details = await Promise.all(forms.map((form) => client.session.form.get({ sessionID, formID: form.id })));
  return details.filter((form) => form.state.status === "pending").map((form) => ({
    id: form.id,
    title: form.title,
    fields: form.fields.map((field) => ({
      key: field.key,
      title: "title" in field ? field.title : undefined,
      description: "description" in field ? field.description : undefined,
      type: field.type,
      custom: "custom" in field ? field.custom ?? false : false,
      options: "options" in field ? (field.options ?? []).map((option) => ({
        value: option.value,
        label: option.label,
        description: option.description,
        recommended: "recommended" in option && option.recommended === true,
      })) : [],
    })),
  }));
}
